"""Pruebas de manual_tests/utils/findings.py (etapa 1: listar, segmento, verificar, contar, ver).

Ejecutar desde la raíz del repositorio:
    python3 -m unittest discover -s manual_tests/utils/tests -v
"""

import contextlib
import io
import os
import shutil
import sys
import tempfile
import unittest

UTILS = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, UTILS)

import findings as F  # noqa: E402

CAMBIO = (
    "## Cambio aplicado\n\n"
    "**Archivos modificados:**\n\n- `services/a.go`\n\n"
    "**Verificación:**\n\nPrueba `services/a_test.go` en verde.\n"
)
DECISION = "## Decisión\n\nSe mantiene el comportamiento. Lo decide el dueño.\n"
PROPUESTA = "## Propuesta\n\nTexto de la propuesta.\n"


def hallazgo(id_="F01", estado="abierto", solucion="propuesta", secciones=None, titulo=None, extra="", gravedad="media"):
    """Devuelve el texto de un finding.md con el formato de utils/README.md."""
    if secciones is None:
        secciones = [PROPUESTA] if solucion in ("ninguna", "propuesta") else [CAMBIO]
        if estado == "aceptado":
            secciones.append(DECISION)
    partes = [
        f"# {id_}: {titulo or 'Título ' + id_}",
        "",
        f"Estado: {estado}. Texto libre.",
        f"Solución: {solucion}. Texto libre.",
        f"Gravedad: {gravedad}. Texto libre.",
        "",
        "## Explicación",
        "",
        "Texto de la explicación.",
        "",
    ]
    partes.extend(secciones)
    return "\n".join(partes) + extra


def crear_arbol(tmp, carpetas, filas=None):
    """Crea findings/ con carpetas {nombre_carpeta: texto} e índice con filas (id, estado, resumen, carpeta).

    Si filas es None, el índice se arma con los datos de cada finding.md (estado tomado de la línea Estado).
    """
    raiz = os.path.join(tmp, "findings")
    os.makedirs(raiz)
    for carpeta, texto in carpetas.items():
        os.makedirs(os.path.join(raiz, carpeta))
        with open(os.path.join(raiz, carpeta, "finding.md"), "w", encoding="utf-8") as f:
            f.write(texto)
    if filas is None:
        filas = []
        for carpeta, texto in sorted(carpetas.items()):
            estado = texto.split("Estado:", 1)[1].split(".", 1)[0].strip() if "Estado:" in texto else "?"
            filas.append((carpeta.split("_")[0], estado, "Resumen", carpeta + "/"))
    lineas = ["# Hallazgos", "", "| ID | Estado | Resumen | Carpeta |", "|---|---|---|---|"]
    for id_, estado, resumen, carpeta in filas:
        lineas.append(f"| {id_} | {estado} | {resumen} | `{carpeta}` |")
    with open(os.path.join(raiz, "findings.md"), "w", encoding="utf-8") as f:
        f.write("\n".join(lineas) + "\n")
    return raiz


def reglas(incumplimientos):
    """Devuelve el conjunto de pares (hallazgo, regla) de una lista de incumplimientos."""
    return {(i.hallazgo, i.regla) for i in incumplimientos}


class Base(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.mkdtemp()
        self.addCleanup(shutil.rmtree, self.tmp, True)


class VerificarArbol(Base):
    def test_verificar_tabla_imprime_resumen_de_secciones(self):
        raiz = crear_arbol(self.tmp, {"F01_uno": hallazgo("F01", "abierto", "propuesta")})
        out, err = io.StringIO(), io.StringIO()
        with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
            F.main(["verificar"], raiz=raiz)
        self.assertIn("F01", out.getvalue())
        self.assertIn("Explicación=sí", out.getvalue())

    def test_arbol_valido_no_tiene_incumplimientos(self):
        raiz = crear_arbol(self.tmp, {
            "F01_uno": hallazgo("F01", "abierto", "propuesta"),
            "F02_dos": hallazgo("F02", "aceptado", "ninguna"),
            "F03_tres": hallazgo("F03", "abierto", "aplicada_sin_commit"),
        })
        self.assertEqual(F.verificar(raiz), [])

    def test_indice_con_id_sin_carpeta_se_reporta(self):
        raiz = crear_arbol(self.tmp, {"F01_uno": hallazgo("F01")},
                           filas=[("F01", "abierto", "R", "F01_uno/"), ("F09", "abierto", "R", "F09_x/")])
        self.assertIn(("F09", "indice_sin_carpeta"), reglas(F.verificar(raiz)))

    def test_carpeta_sin_fila_en_indice_se_reporta(self):
        raiz = crear_arbol(self.tmp, {"F01_uno": hallazgo("F01"), "F02_dos": hallazgo("F02")},
                           filas=[("F01", "abierto", "R", "F01_uno/")])
        self.assertIn(("F02", "carpeta_sin_indice"), reglas(F.verificar(raiz)))

    def test_estado_del_indice_distinto_del_archivo_se_reporta(self):
        raiz = crear_arbol(self.tmp, {"F01_uno": hallazgo("F01", "abierto")},
                           filas=[("F01", "resuelto", "R", "F01_uno/")])
        self.assertIn(("F01", "indice_estado_distinto"), reglas(F.verificar(raiz)))


class VerificarContenido(Base):
    def test_titulo_con_id_distinto_de_la_carpeta_se_reporta(self):
        raiz = crear_arbol(self.tmp, {"F01_uno": hallazgo("F09", "abierto")})
        self.assertIn(("F01", "titulo"), reglas(F.verificar(raiz)))

    def test_estado_fuera_de_vocabulario_se_reporta(self):
        texto = hallazgo("F01").replace("Estado: abierto.", "Estado: corregido en el árbol de trabajo.")
        raiz = crear_arbol(self.tmp, {"F01_uno": texto},
                           filas=[("F01", "abierto", "R", "F01_uno/")])
        self.assertIn(("F01", "estado_vocabulario"), reglas(F.verificar(raiz)))

    def test_falta_linea_solucion_se_reporta(self):
        texto = hallazgo("F01").replace("Solución: propuesta. Texto libre.\n", "")
        raiz = crear_arbol(self.tmp, {"F01_uno": texto})
        self.assertIn(("F01", "solucion_vocabulario"), reglas(F.verificar(raiz)))

    def test_falta_explicacion_se_reporta(self):
        texto = hallazgo("F01").replace("## Explicación", "## Otra")
        raiz = crear_arbol(self.tmp, {"F01_uno": texto})
        self.assertIn(("F01", "seccion_explicacion"), reglas(F.verificar(raiz)))

    def test_solucion_propuesta_exige_seccion_propuesta(self):
        texto = hallazgo("F01", "abierto", "propuesta", secciones=[])
        raiz = crear_arbol(self.tmp, {"F01_uno": texto})
        self.assertIn(("F01", "seccion_propuesta"), reglas(F.verificar(raiz)))

    def test_solucion_aplicada_exige_cambio_aplicado(self):
        texto = hallazgo("F01", "abierto", "aplicada_sin_commit", secciones=[PROPUESTA])
        raiz = crear_arbol(self.tmp, {"F01_uno": texto})
        self.assertIn(("F01", "seccion_cambio_aplicado"), reglas(F.verificar(raiz)))

    def test_cambio_aplicado_sin_verificacion_se_reporta(self):
        cambio = "## Cambio aplicado\n\n**Archivos modificados:**\n\n- `a.go`\n"
        texto = hallazgo("F01", "abierto", "aplicada_sin_commit", secciones=[cambio])
        raiz = crear_arbol(self.tmp, {"F01_uno": texto})
        self.assertIn(("F01", "cambio_aplicado_contenido"), reglas(F.verificar(raiz)))

    def test_resuelto_exige_cambio_aplicado(self):
        texto = hallazgo("F01", "resuelto", "aplicada_sin_commit", secciones=[PROPUESTA])
        raiz = crear_arbol(self.tmp, {"F01_uno": texto})
        self.assertIn(("F01", "seccion_cambio_aplicado"), reglas(F.verificar(raiz)))

    def test_aceptado_exige_decision(self):
        texto = hallazgo("F01", "aceptado", "ninguna", secciones=[PROPUESTA])
        raiz = crear_arbol(self.tmp, {"F01_uno": texto})
        self.assertIn(("F01", "seccion_decision"), reglas(F.verificar(raiz)))

    def test_encabezado_repetido_se_reporta(self):
        texto = hallazgo("F01", secciones=[PROPUESTA, PROPUESTA])
        raiz = crear_arbol(self.tmp, {"F01_uno": texto})
        self.assertIn(("F01", "encabezado_repetido"), reglas(F.verificar(raiz)))

    def test_bloque_diff_se_reporta(self):
        texto = hallazgo("F01", "abierto", "aplicada_sin_commit",
                         secciones=[CAMBIO + "\n```diff\n+ x\n```\n"])
        raiz = crear_arbol(self.tmp, {"F01_uno": texto})
        self.assertIn(("F01", "bloque_diff"), reglas(F.verificar(raiz)))

    def test_id_de_carpeta_con_formato_invalido_se_reporta(self):
        raiz = crear_arbol(self.tmp, {"F1_uno": hallazgo("F1")}, filas=[])
        self.assertIn(("F1_uno", "id_formato"), reglas(F.verificar(raiz)))

    def test_verificar_devuelve_codigo_1_con_incumplimientos_y_0_sin_ellos(self):
        valido = crear_arbol(self.tmp, {"F01_uno": hallazgo("F01")})
        out, err = io.StringIO(), io.StringIO()
        with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
            self.assertEqual(F.main(["verificar"], raiz=valido), 0)
        malo_dir = tempfile.mkdtemp()
        self.addCleanup(shutil.rmtree, malo_dir, True)
        malo = crear_arbol(malo_dir, {"F01_uno": hallazgo("F09")})
        with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
            self.assertEqual(F.main(["verificar"], raiz=malo), 1)


    def test_resumen_indica_secciones_presentes_y_no_aplicables(self):
        raiz = crear_arbol(self.tmp, {"F01_uno": hallazgo("F01", "abierto", "propuesta")})
        resumen = F.resumen_secciones(raiz)
        self.assertTrue(resumen["F01"]["Propuesta"])
        self.assertIsNone(resumen["F01"]["Cambio aplicado"])


class VerificarNotas(Base):
    """Sección opcional `## Notas de especificación`: aparece en el resumen y no puede estar vacía."""

    def test_notas_ausentes_no_producen_incumplimiento_ni_marca(self):
        raiz = crear_arbol(self.tmp, {"F01_uno": hallazgo("F01", "abierto", "propuesta")})
        self.assertNotIn((("F01", "notas_vacias")), reglas(F.verificar(raiz)))
        self.assertIsNone(F.resumen_secciones(raiz)["F01"]["Notas de especificación"])

    def test_notas_presentes_se_marcan_en_el_resumen(self):
        notas = "## Notas de especificación\n\nConflicto con openspec/specs/proveedor-write.\n"
        raiz = crear_arbol(self.tmp, {"F01_uno": hallazgo("F01", "abierto", "propuesta", secciones=[PROPUESTA, notas])})
        self.assertTrue(F.resumen_secciones(raiz)["F01"]["Notas de especificación"])
        self.assertEqual(reglas(F.verificar(raiz)), set())

    def test_notas_vacias_se_reportan(self):
        notas = "## Notas de especificación\n\n"
        raiz = crear_arbol(self.tmp, {"F01_uno": hallazgo("F01", "abierto", "propuesta", secciones=[PROPUESTA, notas])})
        self.assertIn(("F01", "notas_vacias"), reglas(F.verificar(raiz)))


class Gravedad(Base):
    """Etiqueta `Gravedad:` obligatoria, con vocabulario fijo, y su cambio y filtro."""

    def test_gravedad_ausente_se_reporta(self):
        texto = hallazgo("F01").replace("Gravedad: media. Texto libre.\n", "")
        raiz = crear_arbol(self.tmp, {"F01_uno": texto})
        self.assertIn(("F01", "gravedad_vocabulario"), reglas(F.verificar(raiz)))

    def test_gravedad_fuera_de_vocabulario_se_reporta(self):
        texto = hallazgo("F01", gravedad="muy alta")
        raiz = crear_arbol(self.tmp, {"F01_uno": texto})
        self.assertIn(("F01", "gravedad_vocabulario"), reglas(F.verificar(raiz)))

    def test_gravedad_valida_no_produce_incumplimiento(self):
        raiz = crear_arbol(self.tmp, {"F01_uno": hallazgo("F01", gravedad="critica")})
        self.assertEqual(reglas(F.verificar(raiz)), set())

    def test_cambiar_gravedad_actualiza_la_linea(self):
        raiz = crear_arbol(self.tmp, {"F01_uno": hallazgo("F01")})
        F.cambiar(raiz, "F01", gravedad="alta", aplicar=True)
        with open(os.path.join(raiz, "F01_uno", "finding.md"), encoding="utf-8") as f:
            self.assertIn("Gravedad: alta.", f.read())

    def test_cambiar_gravedad_fuera_de_vocabulario_es_uso_incorrecto(self):
        raiz = crear_arbol(self.tmp, {"F01_uno": hallazgo("F01")})
        with self.assertRaises(F.ErrorUso):
            F.cambiar(raiz, "F01", gravedad="grave", aplicar=True)

    def test_listar_filtra_por_gravedad(self):
        raiz = crear_arbol(self.tmp, {
            "F01_uno": hallazgo("F01", gravedad="alta"),
            "F02_dos": hallazgo("F02", gravedad="baja"),
        })
        ids = [r["id"] for r in F.listar(raiz, gravedad="alta")]
        self.assertEqual(ids, ["F01"])

    def test_listar_gravedad_fuera_de_vocabulario_es_uso_incorrecto(self):
        raiz = crear_arbol(self.tmp, {"F01_uno": hallazgo("F01")})
        with self.assertRaises(F.ErrorUso):
            F.listar(raiz, gravedad="grave")


class Segmento(Base):
    def test_segmento_incluye_subencabezados_y_se_detiene_en_mismo_nivel(self):
        texto = hallazgo("F01", secciones=[
            "## Causa\n\nC1.\n\n### Detalle\n\nD1.\n\n## Propuesta\n\nP1.\n"
        ])
        raiz = crear_arbol(self.tmp, {"F01_uno": texto})
        seg = F.segmento(raiz, "F01", encabezado="## Causa")
        self.assertIn("C1.", seg)
        self.assertIn("### Detalle", seg)
        self.assertIn("D1.", seg)
        self.assertNotIn("## Propuesta", seg)
        self.assertNotIn("P1.", seg)

    def test_encabezado_inexistente_es_error_de_validacion(self):
        raiz = crear_arbol(self.tmp, {"F01_uno": hallazgo("F01")})
        with self.assertRaises(F.ErrorValidacion):
            F.segmento(raiz, "F01", encabezado="## No existe")

    def test_encabezado_ambiguo_lista_las_coincidencias(self):
        texto = hallazgo("F01", secciones=[PROPUESTA, "## Propuesta\n\nOtra.\n"])
        raiz = crear_arbol(self.tmp, {"F01_uno": texto})
        with self.assertRaises(F.ErrorValidacion) as ctx:
            F.segmento(raiz, "F01", encabezado="## Propuesta")
        self.assertIn("2", str(ctx.exception))

    def test_rango_de_lineas_fuera_del_archivo_es_error(self):
        raiz = crear_arbol(self.tmp, {"F01_uno": hallazgo("F01")})
        with open(os.path.join(raiz, "F01_uno", "finding.md"), encoding="utf-8") as f:
            total = len(f.read().splitlines())
        with self.assertRaises(F.ErrorValidacion):
            F.segmento(raiz, "F01", lineas=(1, total + 5))

    def test_rango_de_lineas_devuelve_las_lineas_pedidas(self):
        raiz = crear_arbol(self.tmp, {"F01_uno": hallazgo("F01")})
        seg = F.segmento(raiz, "F01", lineas=(1, 1))
        self.assertEqual(seg.strip(), "# F01: Título F01")

    def test_id_con_ruta_fuera_de_findings_es_uso_incorrecto(self):
        raiz = crear_arbol(self.tmp, {"F01_uno": hallazgo("F01")})
        with self.assertRaises(F.ErrorUso):
            F.segmento(raiz, "../F01", lineas=(1, 1))


class ListarContarVer(Base):
    def setUp(self):
        super().setUp()
        self.raiz = crear_arbol(self.tmp, {
            "F02_dos": hallazgo("F02", "resuelto", "aplicada_sin_commit"),
            "F01_uno": hallazgo("F01", "abierto", "propuesta"),
            "F03_tres": hallazgo("F03", "abierto", "ninguna"),
        })

    def test_listar_ordena_por_id_y_filtra_por_estado(self):
        todos = [r["id"] for r in F.listar(self.raiz)]
        self.assertEqual(todos, ["F01", "F02", "F03"])
        abiertos = [r["id"] for r in F.listar(self.raiz, estado="abierto")]
        self.assertEqual(abiertos, ["F01", "F03"])

    def test_listar_filtra_por_solucion(self):
        ids = [r["id"] for r in F.listar(self.raiz, solucion="propuesta")]
        self.assertEqual(ids, ["F01"])

    def test_contar_devuelve_totales_por_estado_y_solucion(self):
        c = F.contar(self.raiz)
        self.assertEqual(c["por_estado"], {"abierto": 2, "resuelto": 1})
        self.assertEqual(c["por_solucion"], {"propuesta": 1, "ninguna": 1, "aplicada_sin_commit": 1})

    def test_ver_devuelve_campos_y_encabezados(self):
        v = F.ver(self.raiz, "F01")
        self.assertEqual(v["estado"], "abierto")
        self.assertEqual(v["solucion"], "propuesta")
        self.assertEqual(v["titulo"], "Título F01")
        self.assertIn("## Explicación", v["encabezados"])
        self.assertIn("## Propuesta", v["encabezados"])

    def test_ver_id_inexistente_es_error_de_validacion(self):
        with self.assertRaises(F.ErrorValidacion):
            F.ver(self.raiz, "F99")

    def test_id_invalido_es_uso_incorrecto(self):
        for malo in ["F1", "f01", "F01x", "../F01"]:
            with self.subTest(valor=malo):
                with self.assertRaises(F.ErrorUso):
                    F.validar_id(malo)


class Cambiar(Base):
    def setUp(self):
        super().setUp()
        self.raiz = crear_arbol(self.tmp, {
            "F01_uno": hallazgo("F01", "abierto", "propuesta"),
            "F02_dos": hallazgo("F02", "abierto", "ninguna"),
        })
        self.ruta = os.path.join(self.raiz, "F01_uno", "finding.md")
        self.indice = os.path.join(self.raiz, "findings.md")

    def leer(self, ruta):
        with open(ruta, encoding="utf-8") as f:
            return f.read()

    def test_sin_aplicar_no_escribe_y_describe_el_cambio(self):
        antes = (self.leer(self.ruta), self.leer(self.indice))
        texto = F.cambiar(self.raiz, "F01", estado="en_decision")
        self.assertEqual((self.leer(self.ruta), self.leer(self.indice)), antes)
        self.assertIn("en_decision", texto)

    def test_aplicar_cambia_solo_la_linea_de_estado_y_la_celda_del_indice(self):
        antes_f, antes_i = self.leer(self.ruta), self.leer(self.indice)
        F.cambiar(self.raiz, "F01", estado="en_decision", aplicar=True)
        self.assertEqual(self.leer(self.ruta), antes_f.replace("Estado: abierto.", "Estado: en_decision.", 1))
        self.assertEqual(self.leer(self.indice), antes_i.replace("| F01 | abierto |", "| F01 | en_decision |", 1))

    def test_cambio_de_solucion_actualiza_la_linea_solucion(self):
        F.cambiar(self.raiz, "F01", solucion="ninguna", aplicar=True)
        self.assertIn("Solución: ninguna.", self.leer(self.ruta))
        self.assertNotIn("Solución: propuesta.", self.leer(self.ruta))

    def test_valor_fuera_de_vocabulario_es_uso_incorrecto(self):
        with self.assertRaises(F.ErrorUso):
            F.cambiar(self.raiz, "F01", estado="listo")

    def test_sin_estado_ni_solucion_es_uso_incorrecto(self):
        with self.assertRaises(F.ErrorUso):
            F.cambiar(self.raiz, "F01")

    def test_resuelto_sin_cambio_aplicado_se_rechaza_sin_escribir(self):
        antes = (self.leer(self.ruta), self.leer(self.indice))
        with self.assertRaises(F.ErrorValidacion):
            F.cambiar(self.raiz, "F01", estado="resuelto", aplicar=True)
        self.assertEqual((self.leer(self.ruta), self.leer(self.indice)), antes)

    def test_aceptado_sin_decision_se_rechaza(self):
        with self.assertRaises(F.ErrorValidacion):
            F.cambiar(self.raiz, "F01", estado="aceptado", aplicar=True)

    def test_solucion_aplicada_sin_cambio_aplicado_se_rechaza(self):
        with self.assertRaises(F.ErrorValidacion):
            F.cambiar(self.raiz, "F02", solucion="aplicada_sin_commit", aplicar=True)

    def test_resuelto_con_cambio_aplicado_completo_se_acepta(self):
        carpeta = os.path.join(self.raiz, "F03_tres")
        os.makedirs(carpeta)
        with open(os.path.join(carpeta, "finding.md"), "w", encoding="utf-8") as f:
            f.write(hallazgo("F03", "abierto", "aplicada_sin_commit"))
        with open(self.indice, "a", encoding="utf-8") as f:
            f.write("| F03 | abierto | R | `F03_tres/` |\n")
        F.cambiar(self.raiz, "F03", estado="resuelto", aplicar=True)
        self.assertIn("Estado: resuelto.", self.leer(os.path.join(carpeta, "finding.md")))

    def test_fila_de_indice_inexistente_se_rechaza(self):
        raiz = crear_arbol(tempfile.mkdtemp(dir=self.tmp), {"F01_uno": hallazgo("F01")},
                           filas=[("F02", "abierto", "R", "F02_x/")])
        self.addCleanup(shutil.rmtree, os.path.dirname(raiz), True)
        with self.assertRaises(F.ErrorValidacion):
            F.cambiar(raiz, "F01", estado="en_decision", aplicar=True)

    def test_no_deja_archivos_temporales_en_la_carpeta(self):
        F.cambiar(self.raiz, "F01", estado="en_decision", aplicar=True)
        self.assertEqual(sorted(os.listdir(os.path.join(self.raiz, "F01_uno"))), ["finding.md"])
        self.assertEqual(sorted(os.listdir(self.raiz)), ["F01_uno", "F02_dos", "findings.md"])

    def test_id_inexistente_es_error_de_validacion(self):
        with self.assertRaises(F.ErrorValidacion):
            F.cambiar(self.raiz, "F99", estado="en_decision")

    def test_main_cambiar_devuelve_codigos_de_salida(self):
        out, err = io.StringIO(), io.StringIO()
        with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
            self.assertEqual(F.main(["cambiar", "F01", "--estado", "listo"], raiz=self.raiz), 2)
            self.assertEqual(F.main(["cambiar", "F01", "--estado", "en_decision"], raiz=self.raiz), 0)
            self.assertEqual(F.main(["cambiar", "F01", "--estado", "resuelto", "--aplicar"], raiz=self.raiz), 1)
        self.assertIn("Estado: abierto.", self.leer(self.ruta))


class HallazgosReales(unittest.TestCase):
    def test_hallazgos_reales_cumplen_el_formato(self):
        raiz = os.path.normpath(os.path.join(UTILS, "..", "findings"))
        fallos = F.verificar(raiz)
        self.assertEqual(fallos, [], "\n".join(f"{i.hallazgo} {i.regla}: {i.detalle}" for i in fallos))


if __name__ == "__main__":
    unittest.main()
