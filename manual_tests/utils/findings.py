#!/usr/bin/env python3
"""Gestión de los hallazgos de manual_tests/findings.

Subcomandos: listar, segmento, verificar, contar, ver y cambiar.
La guía de uso, el formato de los hallazgos y la regla de aceptación están en
manual_tests/utils/README.md.

Uso:
    python3 manual_tests/utils/findings.py <subcomando> [opciones]
"""

import argparse
import json
import os
import re
import sys
from collections import namedtuple

RAIZ = os.path.normpath(os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "findings"))

ESTADOS = ("abierto", "en_decision", "resuelto", "aceptado")
SOLUCIONES = ("ninguna", "propuesta", "aplicada_sin_commit")
# Sección opcional (cualquier estado). Contiene notas de contrato para programar cambios.
NOTAS_ESPECIFICACION = "Notas de especificación"
# Gravedad del defecto o diferencia con la especificación. Obligatoria en todo hallazgo.
GRAVEDADES = ("critica", "alta", "media", "baja")

ID_RE = re.compile(r"^F\d{2}$")
CARPETA_RE = re.compile(r"^(F\d{2})_[a-z0-9_]+$")
TITULO_RE = re.compile(r"^# (F\d{2}): (.+)$")
ESTADO_RE = re.compile(r"^Estado: ([^.\n]*)\.", re.M)
SOLUCION_RE = re.compile(r"^Solución: ([^.\n]*)\.", re.M)
GRAVEDAD_RE = re.compile(r"^Gravedad: ([^.\n]*)\.", re.M)
ENCABEZADO_RE = re.compile(r"^(#{1,6}) (.+?)\s*$")

Incumplimiento = namedtuple("Incumplimiento", "hallazgo regla detalle")


class ErrorUso(Exception):
    """Argumentos o identificadores con formato inválido (código de salida 2)."""


class ErrorValidacion(Exception):
    """Hallazgo, encabezado o rango inexistente o incoherente (código de salida 1)."""


def validar_id(valor):
    """Valida que un ID tenga el formato F seguido de dos dígitos.

    Args:
        valor: Texto a validar. Se usa en rutas, por eso el formato es estricto.

    Returns:
        El mismo valor, sin cambios.

    Raises:
        ErrorUso: Si el valor no cumple `F\\d{2}`.
    """
    if not isinstance(valor, str) or not ID_RE.match(valor):
        raise ErrorUso(f"ID inválido: {valor!r}. Formato esperado: F00 a F99.")
    return valor


def _carpetas(raiz):
    """Lista las carpetas de hallazgos de la raíz con su ID.

    Args:
        raiz: Directorio findings/.

    Returns:
        Lista de tuplas (nombre_carpeta, id) de las carpetas que tienen `finding.md`
        o cuyo nombre empieza por F. Las carpetas con nombre no válido se incluyen
        con id None para que `verificar` las reporte.
    """
    salida = []
    for nombre in sorted(os.listdir(raiz)):
        ruta = os.path.join(raiz, nombre)
        if not os.path.isdir(ruta) or not nombre.startswith("F"):
            continue
        coincide = CARPETA_RE.match(nombre)
        salida.append((nombre, coincide.group(1) if coincide else None))
    return salida


def _leer(ruta):
    """Lee un archivo en UTF-8.

    Args:
        ruta: Ruta del archivo.

    Returns:
        Contenido del archivo como texto.
    """
    with open(ruta, encoding="utf-8") as f:
        return f.read()


def _lineas_con_estado_de_codigo(texto):
    """Marca cada línea según esté dentro de un bloque de código delimitado por ```.

    Args:
        texto: Contenido Markdown.

    Returns:
        Lista de tuplas (linea, dentro_de_codigo). La línea de apertura y cierre
        del bloque cuentan como dentro de código.
    """
    salida, dentro = [], False
    for linea in texto.splitlines():
        if linea.lstrip().startswith("```"):
            salida.append((linea, True))
            dentro = not dentro
        else:
            salida.append((linea, dentro))
    return salida


def encabezados(texto):
    """Extrae los encabezados Markdown que no están dentro de un bloque de código.

    Args:
        texto: Contenido Markdown de un hallazgo.

    Returns:
        Lista de tuplas (numero_de_linea, nivel, texto_del_encabezado). El número de
        línea es base 1.
    """
    salida = []
    for n, (linea, en_codigo) in enumerate(_lineas_con_estado_de_codigo(texto), start=1):
        if en_codigo:
            continue
        m = ENCABEZADO_RE.match(linea)
        if m:
            salida.append((n, len(m.group(1)), m.group(2)))
    return salida


def _valor(regex, texto):
    """Devuelve el primer valor capturado por una expresión regular, sin espacios sobrantes.

    Args:
        regex: Expresión compilada con un grupo.
        texto: Texto donde buscar.

    Returns:
        El valor capturado, o None si no aparece.
    """
    m = regex.search(texto)
    return m.group(1).strip() if m else None


def leer_indice(raiz):
    """Lee las filas del índice `findings.md`.

    Args:
        raiz: Directorio findings/.

    Returns:
        Lista de diccionarios con las claves id, estado, resumen y carpeta, en el
        orden del archivo. Las filas de encabezado y separador se omiten.
    """
    filas = []
    for linea in _leer(os.path.join(raiz, "findings.md")).splitlines():
        if not linea.startswith("|"):
            continue
        celdas = [c.strip() for c in linea.strip().strip("|").split("|")]
        if len(celdas) != 4 or celdas[0] in ("ID",) or celdas[0].startswith("-"):
            continue
        filas.append({
            "id": celdas[0],
            "estado": celdas[1],
            "resumen": celdas[2],
            "carpeta": celdas[3].strip("`").rstrip("/"),
        })
    return filas


def _hallazgo(raiz, nombre_carpeta):
    """Carga un hallazgo con los campos derivados de su `finding.md`.

    Args:
        raiz: Directorio findings/.
        nombre_carpeta: Nombre de la carpeta del hallazgo.

    Returns:
        Diccionario de `_hallazgo_desde_texto`.
    """
    ruta = os.path.join(raiz, nombre_carpeta, "finding.md")
    return _hallazgo_desde_texto(nombre_carpeta, _leer(ruta), ruta)


def _hallazgo_desde_texto(nombre_carpeta, texto, ruta):
    """Construye el diccionario de un hallazgo a partir de su texto.

    Args:
        nombre_carpeta: Nombre de la carpeta del hallazgo.
        texto: Contenido de `finding.md`.
        ruta: Ruta del archivo, solo para informar.

    Returns:
        Diccionario con id (de la carpeta), titulo, estado, solucion, texto, ruta
        y secciones (encabezados de nivel 2 sin `## `).
    """
    m = TITULO_RE.match(texto.splitlines()[0]) if texto.strip() else None
    coincide = CARPETA_RE.match(nombre_carpeta)
    return {
        "carpeta": nombre_carpeta,
        "id": coincide.group(1) if coincide else None,
        "titulo": m.group(2) if m else None,
        "estado": _valor(ESTADO_RE, texto),
        "solucion": _valor(SOLUCION_RE, texto),
        "gravedad": _valor(GRAVEDAD_RE, texto),
        "texto": texto,
        "ruta": ruta,
        "secciones": [t for _, nivel, t in encabezados(texto) if nivel == 2],
    }


def _hallazgos(raiz):
    """Carga todos los hallazgos que tienen carpeta con `finding.md`.

    Args:
        raiz: Directorio findings/.

    Returns:
        Lista de diccionarios de `_hallazgo`, ordenada por nombre de carpeta.
    """
    salida = []
    for nombre, _ in _carpetas(raiz):
        if os.path.isfile(os.path.join(raiz, nombre, "finding.md")):
            salida.append(_hallazgo(raiz, nombre))
    return salida


def _requiere_cambio(h):
    """Indica si el hallazgo debe tener la sección `## Cambio aplicado`.

    Args:
        h: Diccionario de `_hallazgo`.

    Returns:
        True si la solución es `aplicada_sin_commit` o el estado es `resuelto`.
    """
    return h["solucion"] == "aplicada_sin_commit" or h["estado"] == "resuelto"


def _marca(valor):
    """Convierte el valor de una sección esperada en un texto corto para la tabla.

    Args:
        valor: True (presente), False (falta) o None (no aplica).

    Returns:
        `sí`, `no` o `-`.
    """
    if valor is None:
        return "-"
    return "sí" if valor else "no"


def _resumen_de(h):
    """Calcula qué secciones se esperan en un hallazgo y si cada una está presente.

    Args:
        h: Diccionario de `_hallazgo`.

    Returns:
        Diccionario con las claves Explicación, Propuesta, Cambio aplicado, Decisión y
        Notas de especificación. El valor es True o False si la sección aplica, y None
        si no aplica. Notas de especificación es opcional: True si está presente, None si no.
    """
    secciones = h["secciones"]
    propuesta_aplica = h["solucion"] in ("ninguna", "propuesta")
    return {
        "Explicación": "Explicación" in secciones,
        "Propuesta": ("Propuesta" in secciones) if propuesta_aplica else None,
        "Cambio aplicado": ("Cambio aplicado" in secciones) if _requiere_cambio(h) else None,
        "Decisión": ("Decisión" in secciones) if h["estado"] == "aceptado" else None,
        NOTAS_ESPECIFICACION: True if NOTAS_ESPECIFICACION in secciones else None,
    }


def resumen_secciones(raiz):
    """Resume, por hallazgo, las secciones esperadas y su presencia.

    Args:
        raiz: Directorio findings/.

    Returns:
        Diccionario {id: {seccion: bool o None}}. Ver `_resumen_de`.
    """
    return {h["id"]: _resumen_de(h) for h in _hallazgos(raiz) if h["id"]}


def _incumplimientos_hallazgo(h):
    """Revisa un hallazgo contra el formato de utils/README.md.

    Args:
        h: Diccionario de `_hallazgo`.

    Returns:
        Lista de Incumplimiento. Vacía si el hallazgo cumple el formato.
    """
    nombre = h["carpeta"]
    id_ = h["id"] or nombre
    fallos = []

    def falla(regla, detalle):
        fallos.append(Incumplimiento(id_, regla, detalle))

    if h["id"] is None:
        falla("id_formato", f"La carpeta {nombre} no cumple <ID>_<slug>.")
    if h["titulo"] is None or (h["id"] and TITULO_RE.match(h["texto"].splitlines()[0]).group(1) != h["id"]):
        falla("titulo", "La línea 1 debe ser `# <ID>: <título>` con el ID de la carpeta.")
    if h["estado"] not in ESTADOS:
        falla("estado_vocabulario", f"Estado {h['estado']!r} no está en el vocabulario.")
    if h["solucion"] not in SOLUCIONES:
        falla("solucion_vocabulario", f"Solución {h['solucion']!r} no está en el vocabulario.")
    if h["gravedad"] not in GRAVEDADES:
        falla("gravedad_vocabulario", f"Gravedad {h['gravedad']!r} no está en el vocabulario (critica, alta, media, baja).")
    repetidos = {s for s in h["secciones"] if h["secciones"].count(s) > 1}
    for s in sorted(repetidos):
        falla("encabezado_repetido", f"El encabezado `## {s}` aparece más de una vez.")
    if "Explicación" not in h["secciones"]:
        falla("seccion_explicacion", "Falta `## Explicación`.")
    if h["solucion"] in ("ninguna", "propuesta") and "Propuesta" not in h["secciones"]:
        falla("seccion_propuesta", "Falta `## Propuesta` (solución ninguna o propuesta).")
    if _requiere_cambio(h):
        if "Cambio aplicado" not in h["secciones"]:
            falla("seccion_cambio_aplicado", "Falta `## Cambio aplicado`.")
        else:
            bloque = _seccion(h["texto"], "Cambio aplicado")
            if "**Archivos modificados:**" not in bloque or "**Verificación:**" not in bloque:
                falla("cambio_aplicado_contenido",
                      "`## Cambio aplicado` debe tener **Archivos modificados:** y **Verificación:**.")
    if h["estado"] == "aceptado" and "Decisión" not in h["secciones"]:
        falla("seccion_decision", "Falta `## Decisión` (estado aceptado).")
    if NOTAS_ESPECIFICACION in h["secciones"]:
        contenido = _seccion(h["texto"], NOTAS_ESPECIFICACION).splitlines()[1:]
        if not "".join(contenido).strip():
            falla("notas_vacias", f"`## {NOTAS_ESPECIFICACION}` no puede estar vacía.")
    if any(linea.startswith("```diff") for linea, _ in _lineas_con_estado_de_codigo(h["texto"]) if linea):
        falla("bloque_diff", "No se permiten bloques diff. Describir los archivos en `## Cambio aplicado`.")
    return fallos


def _seccion(texto, titulo_seccion):
    """Devuelve el contenido de una sección de nivel 2 (sin el encabezado).

    Args:
        texto: Contenido Markdown del hallazgo.
        titulo_seccion: Texto del encabezado de nivel 2, sin `## `.

    Returns:
        Texto entre el encabezado y el siguiente `##`, o cadena vacía si no existe.
    """
    lineas = texto.splitlines()
    encs = encabezados(texto)
    for idx, (n, nivel, t) in enumerate(encs):
        if nivel == 2 and t == titulo_seccion:
            fin = len(lineas)
            for n2, nivel2, _ in encs[idx + 1:]:
                if nivel2 <= 2:
                    fin = n2 - 1
                    break
            return "\n".join(lineas[n - 1:fin])
    return ""


def verificar(raiz):
    """Revisa el índice y todos los hallazgos contra el formato de utils/README.md.

    Args:
        raiz: Directorio findings/.

    Returns:
        Lista de Incumplimiento (hallazgo, regla, detalle), ordenada por hallazgo.
    """
    fallos = []
    hallazgos = _hallazgos(raiz)
    for h in hallazgos:
        fallos.extend(_incumplimientos_hallazgo(h))

    por_carpeta = {h["carpeta"]: h for h in hallazgos}
    for nombre, id_ in _carpetas(raiz):
        if id_ and nombre not in por_carpeta:
            fallos.append(Incumplimiento(id_, "archivo_faltante", f"{nombre} no tiene finding.md."))

    filas = leer_indice(raiz)
    ids_indice = set()
    for fila in filas:
        id_ = fila["id"]
        ids_indice.add(id_)
        if not ID_RE.match(id_):
            fallos.append(Incumplimiento(id_, "id_formato", "El ID del índice no cumple F\\d{2}."))
            continue
        coincidentes = [h for h in hallazgos if h["id"] == id_]
        if not coincidentes:
            fallos.append(Incumplimiento(id_, "indice_sin_carpeta", "El índice lista el ID sin carpeta."))
            continue
        h = coincidentes[0]
        if fila["estado"] != h["estado"]:
            fallos.append(Incumplimiento(
                id_, "indice_estado_distinto",
                f"Índice: {fila['estado']!r}. finding.md: {h['estado']!r}."))

    for h in hallazgos:
        if h["id"] and h["id"] not in ids_indice:
            fallos.append(Incumplimiento(h["id"], "carpeta_sin_indice", "La carpeta no tiene fila en el índice."))

    return sorted(fallos, key=lambda i: (i.hallazgo, i.regla))


def listar(raiz, estado=None, solucion=None, gravedad=None):
    """Lista los hallazgos con su estado y su solución, filtrados si se indica.

    Args:
        raiz: Directorio findings/.
        estado: Valor de estado para filtrar, o None.
        solucion: Valor de solución para filtrar, o None.
        gravedad: Valor de gravedad para filtrar, o None.

    Returns:
        Lista de diccionarios con id, estado, solucion, gravedad, titulo y ruta, ordenada por ID.

    Raises:
        ErrorUso: Si el estado o la solución no están en el vocabulario.
    """
    if estado is not None and estado not in ESTADOS:
        raise ErrorUso(f"Estado {estado!r} fuera del vocabulario.")
    if solucion is not None and solucion not in SOLUCIONES:
        raise ErrorUso(f"Solución {solucion!r} fuera del vocabulario.")
    if gravedad is not None and gravedad not in GRAVEDADES:
        raise ErrorUso(f"Gravedad {gravedad!r} fuera del vocabulario.")
    salida = []
    for h in _hallazgos(raiz):
        if h["id"] is None:
            continue
        if estado is not None and h["estado"] != estado:
            continue
        if solucion is not None and h["solucion"] != solucion:
            continue
        if gravedad is not None and h["gravedad"] != gravedad:
            continue
        salida.append({"id": h["id"], "estado": h["estado"], "solucion": h["solucion"],
                       "gravedad": h["gravedad"], "titulo": h["titulo"], "ruta": h["ruta"]})
    return sorted(salida, key=lambda r: r["id"])


def contar(raiz):
    """Cuenta los hallazgos por estado, por solución y por la combinación de ambos.

    Args:
        raiz: Directorio findings/.

    Returns:
        Diccionario con las claves por_estado, por_solucion (conteos) y cruce
        (conteos con clave `estado/solucion`).
    """
    por_estado, por_solucion, cruce = {}, {}, {}
    for h in _hallazgos(raiz):
        if h["id"] is None:
            continue
        por_estado[h["estado"]] = por_estado.get(h["estado"], 0) + 1
        por_solucion[h["solucion"]] = por_solucion.get(h["solucion"], 0) + 1
        clave = f"{h['estado']}/{h['solucion']}"
        cruce[clave] = cruce.get(clave, 0) + 1
    return {"por_estado": por_estado, "por_solucion": por_solucion, "cruce": cruce}


def _buscar(raiz, id_):
    """Devuelve el hallazgo con el ID dado.

    Args:
        raiz: Directorio findings/.
        id_: ID ya validado.

    Returns:
        Diccionario de `_hallazgo`.

    Raises:
        ErrorValidacion: Si no hay carpeta con ese ID.
    """
    for h in _hallazgos(raiz):
        if h["id"] == id_:
            return h
    raise ErrorValidacion(f"No hay hallazgo {id_} en {raiz}.")


def ver(raiz, id_):
    """Devuelve los campos de un hallazgo y su lista de encabezados de nivel 2.

    Args:
        raiz: Directorio findings/.
        id_: ID del hallazgo.

    Returns:
        Diccionario con id, estado, solucion, titulo, ruta y encabezados (lista de
        `## <texto>`).

    Raises:
        ErrorUso: Si el ID tiene formato inválido.
        ErrorValidacion: Si el hallazgo no existe.
    """
    validar_id(id_)
    h = _buscar(raiz, id_)
    return {"id": id_, "estado": h["estado"], "solucion": h["solucion"], "titulo": h["titulo"],
            "ruta": h["ruta"], "encabezados": [f"## {s}" for s in h["secciones"]]}


def segmento(raiz, id_, encabezado=None, lineas=None):
    """Recupera un segmento de un hallazgo por encabezado o por rango de líneas.

    Args:
        raiz: Directorio findings/.
        id_: ID del hallazgo.
        encabezado: Encabezado completo (por ejemplo `## Causa`), o None.
        lineas: Tupla (inicio, fin) base 1 e inclusiva, o None.

    Returns:
        Texto del segmento. Con encabezado, incluye los subencabezados y termina
        antes del siguiente encabezado de igual o menor nivel.

    Raises:
        ErrorUso: Si el ID es inválido o no se indica exactamente un selector.
        ErrorValidacion: Si el encabezado no existe o aparece varias veces, o si el
            rango sale del archivo.
    """
    validar_id(id_)
    if (encabezado is None) == (lineas is None):
        raise ErrorUso("Indicar exactamente uno de --encabezado o --lineas.")
    h = _buscar(raiz, id_)
    todas = h["texto"].splitlines()

    if lineas is not None:
        inicio, fin = lineas
        if inicio < 1 or fin > len(todas) or inicio > fin:
            raise ErrorValidacion(f"Rango {inicio}:{fin} fuera de 1..{len(todas)}.")
        return "\n".join(todas[inicio - 1:fin])

    nivel = len(encabezado) - len(encabezado.lstrip("#"))
    texto_buscado = encabezado.lstrip("#").strip()
    coincidencias = [(n, nv) for n, nv, t in encabezados(h["texto"]) if nv == nivel and t == texto_buscado]
    if not coincidencias:
        raise ErrorValidacion(f"No existe el encabezado {encabezado!r} en {id_}.")
    if len(coincidencias) > 1:
        lista = ", ".join(str(n) for n, _ in coincidencias)
        raise ErrorValidacion(f"El encabezado {encabezado!r} aparece {len(coincidencias)} veces "
                              f"(líneas {lista}).")
    inicio = coincidencias[0][0]
    fin = len(todas)
    for n, nv, _ in encabezados(h["texto"]):
        if n > inicio and nv <= nivel:
            fin = n - 1
            break
    return "\n".join(todas[inicio - 1:fin])


def _reemplazar_linea(texto, regex, etiqueta, valor):
    """Reemplaza el valor de la primera línea `Etiqueta: valor.` de un hallazgo.

    Args:
        texto: Contenido de `finding.md`.
        regex: Expresión compilada que captura el valor actual.
        etiqueta: Texto de la etiqueta (`Estado` o `Solución`).
        valor: Nuevo valor, ya validado contra el vocabulario.

    Returns:
        Texto con la línea reemplazada. El resto del archivo no cambia.

    Raises:
        ErrorValidacion: Si el hallazgo no tiene la línea.
    """
    nuevo, n = regex.subn(lambda _: f"{etiqueta}: {valor}.", texto, count=1)
    if n == 0:
        raise ErrorValidacion(f"Falta la línea `{etiqueta}:` en el hallazgo.")
    return nuevo


def _reemplazar_estado_indice(texto_indice, id_, estado):
    """Reemplaza la celda de estado de la fila del ID en el índice.

    Args:
        texto_indice: Contenido de `findings.md`.
        id_: ID validado del hallazgo.
        estado: Nuevo valor de estado.

    Returns:
        Texto del índice con la celda cambiada. El resto de las filas no cambia.

    Raises:
        ErrorValidacion: Si el índice no tiene fila para el ID.
    """
    patron = re.compile(rf"^(\|\s*{id_}\s*\|)[^|\n]*(\|)", re.M)
    nuevo, n = patron.subn(lambda m: f"{m.group(1)} {estado} {m.group(2)}", texto_indice, count=1)
    if n == 0:
        raise ErrorValidacion(f"El índice no tiene fila para {id_}.")
    return nuevo


def _escribir_atomico(ruta, texto):
    """Escribe un archivo reemplazándolo en una sola operación.

    Args:
        ruta: Ruta del archivo destino.
        texto: Contenido completo a escribir.
    """
    temporal = ruta + ".tmp"
    with open(temporal, "w", encoding="utf-8") as f:
        f.write(texto)
    os.replace(temporal, ruta)


def cambiar(raiz, id_, estado=None, solucion=None, aplicar=False, gravedad=None):
    """Cambia el estado o la solución de un hallazgo y la celda de estado del índice.

    Sin `aplicar`, no escribe nada y solo describe el cambio. Con `aplicar`, escribe
    `finding.md` y `findings.md` solo si el resultado cumple el formato de utils/README.md.

    Args:
        raiz: Directorio findings/.
        id_: ID del hallazgo.
        estado: Nuevo estado del vocabulario, o None para no cambiarlo.
        solucion: Nueva solución del vocabulario, o None para no cambiarla.
        aplicar: Si es True, escribe los cambios.

    Returns:
        Texto que describe el cambio y si se aplicó.

    Raises:
        ErrorUso: Si el ID o los valores son inválidos, o no se indica ningún cambio.
        ErrorValidacion: Si el hallazgo o el índice no existen o no tienen la línea
            o la fila necesarias, o si el resultado no cumple el formato.
    """
    validar_id(id_)
    if estado is None and solucion is None and gravedad is None:
        raise ErrorUso("Indicar --estado, --solucion, --gravedad o combinaciones.")
    if estado is not None and estado not in ESTADOS:
        raise ErrorUso(f"Estado {estado!r} fuera del vocabulario.")
    if solucion is not None and solucion not in SOLUCIONES:
        raise ErrorUso(f"Solución {solucion!r} fuera del vocabulario.")
    if gravedad is not None and gravedad not in GRAVEDADES:
        raise ErrorUso(f"Gravedad {gravedad!r} fuera del vocabulario.")

    h = _buscar(raiz, id_)
    nuevo_texto = h["texto"]
    if estado is not None:
        nuevo_texto = _reemplazar_linea(nuevo_texto, ESTADO_RE, "Estado", estado)
    if solucion is not None:
        nuevo_texto = _reemplazar_linea(nuevo_texto, SOLUCION_RE, "Solución", solucion)
    if gravedad is not None:
        nuevo_texto = _reemplazar_linea(nuevo_texto, GRAVEDAD_RE, "Gravedad", gravedad)

    h_nuevo = _hallazgo_desde_texto(h["carpeta"], nuevo_texto, h["ruta"])
    fallos = _incumplimientos_hallazgo(h_nuevo)
    if fallos:
        reglas_fallidas = ", ".join(sorted({f.regla for f in fallos}))
        raise ErrorValidacion(f"El cambio no cumple el formato de utils/README.md: {reglas_fallidas}.")

    ruta_indice = os.path.join(raiz, "findings.md")
    indice_actual = _leer(ruta_indice)
    indice_nuevo = _reemplazar_estado_indice(indice_actual, id_, h_nuevo["estado"])

    partes = []
    if estado is not None:
        partes.append(f"estado {h['estado']} → {h_nuevo['estado']}")
    if solucion is not None:
        partes.append(f"solución {h['solucion']} → {h_nuevo['solucion']}")
    if gravedad is not None:
        partes.append(f"gravedad {h['gravedad']} → {h_nuevo['gravedad']}")
    if aplicar:
        _escribir_atomico(h["ruta"], nuevo_texto)
        if indice_nuevo != indice_actual:
            _escribir_atomico(ruta_indice, indice_nuevo)
        return f"{id_} aplicado: " + "; ".join(partes)
    return f"{id_} propuesto (sin escribir): " + "; ".join(partes)


def _emitir(datos, formato, out):
    """Escribe un resultado como tabla de texto o como JSON.

    Args:
        datos: Estructura a escribir (lista de diccionarios para tabla; dict para JSON).
        formato: `tabla` o `json`.
        out: Flujo de salida.
    """
    if formato == "json":
        out.write(json.dumps(datos, ensure_ascii=False, indent=2) + "\n")
        return
    if isinstance(datos, list):
        for fila in datos:
            out.write("\t".join(str(v) for v in fila.values()) + "\n")
    else:
        for clave, valor in datos.items():
            out.write(f"{clave}: {valor}\n")


def _construir_parser():
    """Crea el parser de argumentos con los subcomandos de utils/README.md.

    Returns:
        argparse.ArgumentParser configurado.
    """
    p = argparse.ArgumentParser(prog="findings.py", description="Gestión de hallazgos de manual_tests/findings.")
    sub = p.add_subparsers(dest="subcomando", required=True)

    s = sub.add_parser("listar", help="Lista hallazgos con estado y solución.")
    s.add_argument("--estado")
    s.add_argument("--solucion")
    s.add_argument("--gravedad")
    s.add_argument("--formato", choices=["tabla", "json"], default="tabla")

    s = sub.add_parser("segmento", help="Recupera un segmento de un hallazgo.")
    s.add_argument("id")
    grupo = s.add_mutually_exclusive_group(required=True)
    grupo.add_argument("--encabezado")
    grupo.add_argument("--lineas", help="Rango inicio:fin, base 1 e inclusivo.")

    s = sub.add_parser("verificar", help="Verifica el índice y los hallazgos contra el formato de utils/README.md.")
    s.add_argument("--formato", choices=["tabla", "json"], default="tabla")

    s = sub.add_parser("contar", help="Cuenta hallazgos por estado y solución.")
    s.add_argument("--formato", choices=["tabla", "json"], default="tabla")

    s = sub.add_parser("ver", help="Muestra los campos y encabezados de un hallazgo.")
    s.add_argument("id")

    s = sub.add_parser("cambiar", help="Cambia estado o solución de un hallazgo (sin --aplicar, solo muestra el cambio).")
    s.add_argument("id")
    s.add_argument("--estado")
    s.add_argument("--solucion")
    s.add_argument("--gravedad")
    s.add_argument("--aplicar", action="store_true")
    return p


def main(argv=None, raiz=RAIZ, out=None, err=None):
    """Punto de entrada del script.

    Args:
        argv: Lista de argumentos, sin el nombre del programa. Si es None, se usa sys.argv.
        raiz: Directorio findings/ sobre el que operar.
        out: Flujo de salida estándar.
        err: Flujo de error.

    Returns:
        Código de salida: 0 correcto, 1 incumplimiento o validación, 2 uso incorrecto.
    """
    out = out or sys.stdout
    err = err or sys.stderr
    try:
        args = _construir_parser().parse_args(argv)
    except SystemExit:
        return 2
    try:
        if args.subcomando == "listar":
            _emitir(listar(raiz, args.estado, args.solucion, args.gravedad), args.formato, out)
            return 0
        if args.subcomando == "segmento":
            lineas = None
            if args.lineas is not None:
                inicio, _, fin = args.lineas.partition(":")
                lineas = (int(inicio), int(fin))
            out.write(segmento(raiz, args.id, encabezado=args.encabezado, lineas=lineas) + "\n")
            return 0
        if args.subcomando == "verificar":
            fallos = verificar(raiz)
            if args.formato == "json":
                datos = {"incumplimientos": [i._asdict() for i in fallos],
                         "secciones": resumen_secciones(raiz)}
                _emitir(datos, "json", out)
            else:
                for i in fallos:
                    out.write(f"{i.hallazgo}\t{i.regla}\t{i.detalle}\n")
                out.write(f"incumplimientos: {len(fallos)}\n")
                for id_, secciones in resumen_secciones(raiz).items():
                    partes = [f"{nombre}={_marca(valor)}" for nombre, valor in secciones.items()]
                    out.write(f"{id_}\tsecciones\t" + " ".join(partes) + "\n")
            return 0 if not fallos else 1
        if args.subcomando == "contar":
            _emitir(contar(raiz), args.formato, out)
            return 0
        if args.subcomando == "ver":
            datos = ver(raiz, args.id)
            _emitir(datos, "tabla", out)
            return 0
        if args.subcomando == "cambiar":
            out.write(cambiar(raiz, args.id, estado=args.estado, solucion=args.solucion,
                              aplicar=args.aplicar, gravedad=args.gravedad) + "\n")
            return 0
    except ErrorUso as e:
        err.write(f"uso incorrecto: {e}\n")
        return 2
    except ErrorValidacion as e:
        err.write(f"error: {e}\n")
        return 1
    return 2


if __name__ == "__main__":
    sys.exit(main())
