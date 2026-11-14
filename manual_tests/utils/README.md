# utils: gestión de hallazgos

`findings.py` es la única vía para consultar y registrar cambios en los hallazgos de `manual_tests/findings/`. Un agente o tester lo usa para leer un hallazgo, buscar un segmento, contar, verificar el formato y cambiar su estado. No sustituye `psql`, `curl` ni el resto del procedimiento de pruebas (`manual_tests/manual_tests.md`).

Leer esta guía antes de registrar o consultar un hallazgo. No abrir los `finding.md` a mano para consultar datos que el script entrega.

## Uso

Desde la raíz del repositorio:

```
python3 manual_tests/utils/findings.py <subcomando> [opciones]
python3 manual_tests/utils/findings.py --help
```

| Subcomando | Para qué sirve | Escribe |
|---|---|---|
| `listar` | Lista hallazgos con estado y solución. Filtros `--estado`, `--solucion`, `--formato tabla\|json`. | No |
| `ver <ID>` | Muestra estado, solución, título, ruta y encabezados `##`. | No |
| `segmento <ID>` | Devuelve una sección por `--encabezado "## Causa"` o un rango con `--lineas 3:10`. | No |
| `verificar` | Revisa el índice y todos los hallazgos contra el formato. Muestra incumplimientos y las secciones presentes por hallazgo. `--formato json` disponible. | No |
| `contar` | Cuenta hallazgos por estado, por solución y por cruce de ambos. | No |
| `cambiar <ID>` | Cambia `--estado` y/o `--solucion`. Sin `--aplicar` solo muestra el cambio propuesto. | Solo con `--aplicar` |

## Flujo de trabajo

1. Consultar antes de tocar: `listar` o `ver`. Para una sección concreta, `segmento`.
2. Registrar o cambiar un hallazgo: primero `cambiar <ID> --estado ... --solucion ...` sin `--aplicar`. Si la salida es correcta, repetir con `--aplicar`.
3. Después de cualquier cambio, ejecutar `verificar`. Debe mostrar `incumplimientos: 0` y salir con código 0.
4. Ejecutar las pruebas del script:

   ```
   python3 -W error -m unittest discover -s manual_tests/utils/tests
   ```

## Resumen de estado

`findings/findings.md` tiene una sección Resumen con la fecha de la última comprobación y los conteos. Se actualiza a mano según el procedimiento de la sección Resumen de `findings/findings.md`. El script no comprueba el Resumen.

## Regla de aceptación

Ningún hallazgo nuevo o modificado se acepta si `verificar` reporta incumplimientos. `cambiar` aplica la misma regla: si el resultado no cumple el formato, no escribe nada. No hay cambios parciales que dejen un hallazgo fuera del formato.

La prueba `HallazgosReales` en `tests/test_findings.py` falla si algún hallazgo del repositorio incumple el formato. Esa prueba forma parte de la verificación del procedimiento.

## Formato de un hallazgo

Carpeta `findings/<ID>_<slug>/` con un `finding.md` (ID en formato `F00` a `F99`, slug en minúsculas, dígitos y guiones bajos):

```
# F03: título del hallazgo

Estado: abierto. Texto libre opcional.
Solución: propuesta. Texto libre opcional.
Gravedad: media. Texto libre opcional.

## Explicación
...
```

Reglas:

- La línea 1 es `# <ID>: <título>`, con el mismo ID que la carpeta.
- Las líneas `Estado:` y `Solución:` van antes del primer `##`. Sus valores están en el vocabulario de abajo.
- `## Explicación` es obligatoria.
- `## Propuesta` es obligatoria si la solución es `ninguna` o `propuesta`.
- `## Cambio aplicado` es obligatoria si la solución es `aplicada_sin_commit` o el estado es `resuelto`. Debe contener `**Archivos modificados:**` con la lista de archivos, y `**Verificación:**` con una explicación breve de cómo comprobar el cambio.
- `## Decisión` es obligatoria si el estado es `aceptado`.
- Los encabezados `##` no se repiten. Otras secciones (Causa, Riesgos, etc.) son libres.
- `## Notas de especificación` es opcional en cualquier estado. Si existe, no puede estar vacía. `verificar` la muestra en la columna `Notas de especificación` (`sí` si está, `-` si no). Sirve para notas de contrato que alguien debe programar: conflicto con `openspec/specs/`, texto propuesto, riesgos verificados y pendientes. Su contenido no se valida más allá de no estar vacía.
- No se usan bloques `diff`. Los cambios se describen con la lista de archivos.

## Vocabulario

Cada hallazgo tiene dos atributos independientes.

Estado:

| Valor | Significado |
|---|---|
| `abierto` | Sin decisión ni solución aplicada. |
| `en_decision` | Requiere una decisión del dueño antes de avanzar. |
| `resuelto` | Corregido y verificado. Requiere `## Cambio aplicado`. |
| `aceptado` | Se decidió mantener el comportamiento. Requiere `## Decisión`. |

Solución:

| Valor | Significado |
|---|---|
| `ninguna` | No hay propuesta técnica cerrada. |
| `propuesta` | Hay propuesta técnica cerrada, no aplicada. |
| `aplicada_sin_commit` | Cambio en el árbol de trabajo. Requiere `## Cambio aplicado`. |

El índice `findings/findings.md` tiene una fila por hallazgo con columnas `ID | Estado | Resumen | Carpeta`. Su estado debe coincidir con el de `finding.md`. `verificar` comprueba esa coincidencia.

Gravedad (obligatoria en todo hallazgo, línea `Gravedad: <valor>.` después de la línea `Solución:`):

| Valor | Significado |
|---|---|
| `critica` | Escribe datos distintos a los pedidos, o responde éxito sin escribir, sin que el cliente lo detecte. |
| `alta` | Persiste datos inválidos, rechaza operaciones válidas, entrega datos incorrectos o impide el arranque. |
| `media` | Incumple la especificación sin pérdida de datos, o produce respuestas confusas. |
| `baja` | Documentación, operación, entorno de pruebas o informativo. |

La gravedad describe el defecto frente a la especificación, no el avance de la corrección. Se asigna con la clasificación del análisis y puede revisarse con `cambiar <ID> --gravedad ...`. `listar --gravedad` filtra por ella.


## Códigos de salida

- `0`: correcto, o `verificar` sin incumplimientos.
- `1`: incumplimiento o validación fallida.
- `2`: uso incorrecto (subcomando u opción desconocidos, ID con formato inválido, valor fuera del vocabulario).

## Alcance

- Opera solo sobre `manual_tests/findings/`. Las rutas que salen de esa carpeta se rechazan.
- Solo `cambiar --aplicar` escribe. Escribe de forma atómica.
- No llama a MID, CRUD, base, terceros ni ubicaciones.
- No lee ni modifica `.env` ni `conf/app.conf`.
- No ejecuta comandos de Git.
- No redacta secciones. Si un hallazgo no cumple el formato, el autor lo corrige a mano y luego vuelve a ejecutar `verificar`.

## Dependencias

Python 3 con la biblioteca estándar. Sin dependencias externas. Las pruebas usan `unittest`.
