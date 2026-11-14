# Pruebas manuales de integración de agora_mid

Este documento describe cómo repetir las pruebas: requisitos, pasos y reglas. No describe el estado de la ejecución. Ese estado vive en otros documentos (ver sección 4): el resumen en `findings/findings.md` y el registro de la última corrida de la matriz en `escritura/resultados.md`.

## Propósito del proceso

Este proceso sirve para identificar errores y proponer sus soluciones. No busca dejar el repositorio en un estado estable. Sigue dos pasos:

1. Identificar y señalar las diferencias entre el sistema y la especificación (`openspec/specs/`, `API.md`). Cada diferencia se registra como hallazgo con su gravedad (`critica`, `alta`, `media` o `baja`, criterios en `utils/README.md`).
2. Proponer e implementar correcciones, indicando los archivos cambiados y las notas sobre la especificación. Cada corrección queda en su hallazgo, en `## Cambio aplicado` y `## Notas de especificación`.

El objetivo es documentar todo para su refinamiento posterior e implementación correcta. Una corrección aplicada sin commit no es definitiva, y la especificación no bloquea ni la identificación ni la corrección. Los textos propuestos para la especificación se aprueban y aplican después, por separado.

## 0. Inicio de sesión asistida

Para iniciar una sesión de pruebas basta con indicar al agente: "sigue `manual_tests/manual_tests.md`, sección 0". Esta sección define el arranque. El resto del documento define los pasos de prueba.

El inicio no escribe en la base, no reinicia servicios y no cambia hallazgos. Solo lee, comprueba y reporta. Cualquier cambio se propone y se confirma con el usuario antes de ejecutarse.

### 0.1 Lectura

1. Este documento completo.
2. `findings/findings.md`: Resumen, Índice y Decisiones pendientes.
3. `utils/README.md`.
4. Las skills indicadas en 3.0, según se vaya a planear o generar código.

### 0.2 Estado de los hallazgos

1. Ejecutar `python3 manual_tests/utils/findings.py verificar` y `contar`. Debe haber `incumplimientos: 0`. Si no, reportarlo y detenerse en este paso.
2. Para cada hallazgo en estado `abierto` o `en_decision`, leer su sección Causa con `findings.py segmento <ID> --encabezado "## Causa"` (o `ver <ID>`).
3. Verificar cada causa contra el código. Ubicar los archivos y funciones que cita la causa, leerlos y comprobar si la afirmación se cumple. Si la causa cita un servicio externo, confirmarla con un `GET` de solo lectura (sección 5).
4. Clasificar cada causa como:
   - **confirmada**: el código o el servicio lo muestran.
   - **parcial**: se cumple en parte. Indicar qué parte.
   - **no confirmada**: el código no lo muestra. Indicar qué se encontró.
   - **obsoleta**: el código cambió y la causa ya no aplica.
5. No cambiar el texto de ningún hallazgo en el inicio. Las correcciones se proponen al usuario.

### 0.3 Estado de las pruebas

1. Comprobar servicios según 3.1 y 3.2 (solo comprobación; no arrancar ni reiniciar sin confirmación).
2. Ejecutar `go build ./...` y `go test -count=1 ./...`. Registrar qué paquetes fallan y a qué hallazgo corresponde cada fallo.
3. Si el usuario lo pide, ejecutar las lecturas de 3.3. Son solo `GET`.
4. Comparar la última corrida de la matriz (`escritura/resultados.md`) con lo que el código y el servicio muestran ahora. Señalar si la corrida ya no es representativa, por ejemplo porque el código cambió después de ella. Para comprobarlo, comparar la fecha de la corrida (`responses/escritura/current/`) con la fecha de modificación de los archivos Go (`git status` y `stat`, sin alterar Git).

### 0.4 Informe de inicio

Entregar al usuario, en este orden:

1. Estado de `verificar`, `contar` y pruebas Go.
2. Por cada hallazgo abierto o en decisión: causa reportada y su clasificación (0.2, punto 4).
3. Estado de servicios y de la última corrida de la matriz.
4. Pasos recomendados, en orden, con lo que requiere confirmación del usuario marcado.

Antes de pasar a la ejecución, acordar con el usuario el alcance: qué hallazgos se atacan, si se repite la matriz y si se toca código.

## 1. Alcance

- Caja negra: `curl` contra la API del middleware (MID) y contra el CRUD `gestion_personas` (lectura de comprobación).
- Servicios:
  - MID `agora_mid`: `http://localhost:8080` (o el puerto de `AGORA_MID_HTTP_PORT`).
  - CRUD `gestion_personas_crud`: `http://localhost:8081/v1`. Su `/swagger/swagger.json` es un stub y no se usa como contrato (F05).
  - Base: contenedor `agora_db` (PostgreSQL 18), esquema `proveedores`, puerto `5432`.
  - Dependencias remotas de solo lectura: terceros (`pruebasapi.intranetoas.udistrital.edu.co:8121`) y ubicaciones (`:8085`, ver F15).
- Escrituras: solo en el esquema `proveedores`, desde la matriz de escritura.

### Responsabilidades

| Componente | Responsable | Qué debe garantizar |
|---|---|---|
| Gestión del MID (arranque, parada, estado, logs y puerto HTTP) | Tester | Que el MID esté desplegado, que su puerto sea accesible desde el entorno de pruebas y que sus logs se puedan leer. Antes de arrancar, verifica si ya hay una instancia en ejecución. Si la hay, pregunta si la reemplaza o si inicia una instancia nueva independiente. Desde que arranca o reemplaza una instancia, es su dueño: puede darla de baja y recargarla según lo requiera la prueba, sin pedir confirmación cada vez. No da de baja una instancia que no arrancó sin preguntar antes. |
| Variables de entorno del MID (`.env`) y `conf/app.conf` | Ambiente | Las carga el ambiente en el proceso del MID. Ningún agente ni el tester las leen ni las modifican. |
| Código fuente del MID | Tester | Es el único componente que el tester modifica. |
| Base `agora_db` (esquema `proveedores`) | Ambiente | Despliegue y disponibilidad en `agora_db`, puerto `5432`. |
| CRUD `gestion_personas` | Ambiente | Despliegue y disponibilidad en `8081`, con su propio `.env`. |
| Terceros, ubicaciones, parámetros | Externos al proyecto | Solo lectura. Su disponibilidad no depende del tester. |

Cualquier cambio en la base, en el CRUD o en el ambiente se solicita al responsable. No se realiza desde estas pruebas.

## 2. Estructura

```
manual_tests/
├── manual_tests.md        este documento (instrucciones)
├── escritura/
│   ├── matriz.md          casos de escritura (POST y PATCH) y efecto esperado
│   └── resultados.md      registro de la última corrida de la matriz
├── requests/              cuerpos JSON de los casos (escritura/ incluye la matriz)
├── responses/
│   ├── current/           salida de scripts/*.sh (lecturas). Se crea al ejecutar esos scripts.
│   └── escritura/current/ última corrida de la matriz (.json y .status)
├── scripts/               una petición por archivo (lecturas en scripts/, matriz en scripts/escritura/)
├── seed/                  dummy_data.sql (carga) y cleanup_dummy_data.sql (sin probar)
├── snapshots/             respaldos de esquema proveedores para restaurar
├── findings/              findings.md (resumen de estado e índice) y una carpeta por hallazgo
└── utils/                 findings.py (gestión de hallazgos). Guía en utils/README.md.
```

## 3. Procedimiento repetible

### 3.0 Lectura previa obligatoria

Antes de ejecutar cualquier paso, el agente o tester lee:

- Este documento completo.
- `findings/findings.md`, sección Resumen: fallos ya conocidos y estado del último registro.
- `utils/README.md`: uso de `findings.py`, formato de los hallazgos, vocabulario y regla de aceptación.
- La skill `test-driven-development` en `.kiro/skills/` antes de planear o generar código. La skill `inline-documentation` antes de tocar código.

No registra ni consulta hallazgos sin haber leído `utils/README.md` en la sesión.

### 3.1 Requisitos

Responsabilidad del ambiente:

- Base `agora_db` arriba y accesible en `5432`.
- CRUD `gestion_personas` en `8081`.

Responsabilidad del tester:

- MID desplegado, con su puerto accesible y sus logs legibles (ver 3.2).

Comprobar antes de escribir:

```shell
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8081/v1/proveedor/1          # 200
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/v1/proveedor/consulta   # 405 esperado (solo POST)
```

Si el CRUD o la base no responden, se avisa al ambiente y no se ejecuta ninguna escritura. Si el MID no responde, el tester lo despliega antes de continuar.

Las variables del MID se cargan con el comando de 3.2 (`set -a && source .env && set +a`). Ningún agente ni el tester leen, copian ni imprimen el contenido de `.env`, ni modifican `conf/app.conf`.

### 3.2 Arranque del middleware

Responsable: tester. Compilar y arrancar el binario. No usar `bee run`, que no termina cuando el binario sale (F11).

Antes de arrancar, comprobar si el MID ya corre:

```shell
pgrep -fa agora_mid_run | grep -v pgrep || echo "sin proceso agora_mid_run"
curl -s -m 3 -o /dev/null -w '%{http_code}\n' http://localhost:8080/v1/proveedor/consulta   # 405 si está arriba
```

Si el MID ya corre, el tester pregunta al usuario si reemplaza esa instancia o si inicia una nueva independiente en otro puerto (`AGORA_MID_HTTP_PORT`, ver abajo). Lo que el usuario elija define la propiedad: si reemplaza, el tester pasa a ser dueño de la instancia; si inicia una nueva, la instancia nueva es suya y la anterior no se toca.

Desde que el tester arranca o reemplaza la instancia, es su dueño. Puede darla de baja (confirmando antes que es el proceso que escucha en el puerto del MID) y volver a arrancarla con el binario actualizado cuando la prueba lo necesite, sin pedir confirmación cada vez. Una instancia que no arrancó el tester no se da de baja sin preguntar.

```shell
cd /home/ndsabogalv/go/src/github.com/udistrital/agora_mid
go build -o /tmp/agora_mid_run .
(set -a && source .env && set +a && nohup /tmp/agora_mid_run > /tmp/agora_mid_run.log 2>&1 &)
```

El tester carga las variables con ese comando y levanta el MID en segundo plano. El log queda en `/tmp/agora_mid_run.log`. Revisarlo tras el arranque (`tail -n 20`). Ningún agente ni el tester imprime ni copia el contenido de `.env`. El arranque consulta el catálogo de terceros (F09). Si falla por timeout, reintentar.

Para probar sin tocar un MID ya en marcha, arrancar en otro puerto: `export AGORA_MID_HTTP_PORT=8090` antes del bloque anterior, y apuntar los scripts a ese puerto (ver 3.4).

### 3.3 Lecturas

```shell
cd manual_tests
for s in scripts/*.sh; do bash "$s"; done
```

Las respuestas quedan en `responses/current/` (`.json` y `.status`). El directorio se crea al ejecutar los scripts. Son lecturas: no escriben. Cada corrida sobrescribe los archivos anteriores.

### 3.4 Matriz de escritura

Antes de escribir, respaldar. El respaldo debe llevar `--clean --if-exists` para poder restaurarse sobre la base viva.

```shell
cd manual_tests
docker exec agora_db pg_dump -U postgres -d postgres -n proveedores --clean --if-exists > snapshots/proveedores_antes_matriz.sql
```

Comprobar la línea base antes de escribir: proveedor 2, natural 1, jurídico 1, contacto 1, cuenta 1 (ver el paso de conteos más abajo). Si no coincide, no escribir y avisar al ambiente.

Ejecutar la matriz. Si el MID está en `8090`, crear copias de los scripts con el puerto cambiado en `/tmp` (no modificar `scripts/`):

```shell
MT=$(pwd)
mkdir -p /tmp/matriz
for s in scripts/escritura/*.sh; do
  sed -e 's#localhost:8080#localhost:8090#' -e "s#^ROOT=.*#ROOT=\"$MT\"#" "$s" > /tmp/matriz/$(basename "$s")
done
for s in /tmp/matriz/*.sh; do bash "$s"; done
```

Con el MID en `8080`, basta `for s in scripts/escritura/*.sh; do bash "$s"; done`. Cada caso escribe `responses/escritura/current/<ID>.json` y `.status`.

Comparar con `escritura/matriz.md` (código HTTP y efecto). Registrar el resultado en `escritura/resultados.md` (ver sección 4).

Restaurar la base y verificar conteos:

```shell
docker exec -i agora_db psql -U postgres -d postgres -v ON_ERROR_STOP=1 -q < snapshots/proveedores_antes_matriz.sql
for t in proveedor proveedor_natural proveedor_juridico contacto cuenta_bancaria; do
  printf "%s=" $t; docker exec agora_db psql -U postgres -At -c "select count(*) from proveedores.$t;"
done
```

Los conteos deben coincidir con la línea base de arriba.

### 3.5 Pruebas Go

```shell
cd /home/ndsabogalv/go/src/github.com/udistrital/agora_mid
go build ./...
go test -count=1 ./...
```

Los fallos conocidos están en `findings/findings.md`. Un fallo que no tenga hallazgo registrado es nuevo: registrarlo con `findings.py` (3.7).

### 3.6 Scripts de carga y limpieza de datos

- Carga: `docker exec -i agora_db psql -U postgres -v ON_ERROR_STOP=1 -f - < seed/dummy_data.sql`. Aborta si ya hay filas en las tablas del esquema.
- Limpieza: `seed/cleanup_dummy_data.sql` borra los ids 1 y 2 de los datos dummy. Probarlo en una base de pruebas antes de usarlo (F13).

### 3.7 Gestión de hallazgos

Usar siempre `utils/findings.py` según `utils/README.md`:

1. Antes de registrar o cambiar un hallazgo, consultarlo con `ver` o `listar`. Para una sección, `segmento`.
2. Registrar o cambiar con `cambiar <ID> --estado ... --solucion ...`. Primero sin `--aplicar` para revisar la propuesta, luego con `--aplicar`.
3. Después de cada cambio, ejecutar `verificar`. Debe mostrar `incumplimientos: 0`.
4. Ejecutar las pruebas del script: `python3 -W error -m unittest discover -s manual_tests/utils/tests`.

Ningún hallazgo nuevo o modificado se acepta con incumplimientos.

### 3.8 Cierre de sesión

Al terminar una sesión que ejecute pruebas o cambie hallazgos:

1. Si hubo corrida de la matriz, reemplazar el contenido de `escritura/resultados.md` con el registro de esa corrida.
2. Actualizar la sección Resumen de `findings/findings.md` con su procedimiento (a mano).
3. Ejecutar `python3 manual_tests/utils/findings.py verificar`.

### 3.9 Corrección rápida y verificación en vivo

Una corrección rápida (arreglo sucio, ver `escritura/matriz.md`, "Arreglos rápidos") se usa solo si el caso bloquea la prueba. Su procedimiento incluye siempre:

1. Pruebas primero, en rojo, y luego la corrección mínima hasta verde (TDD). El marcador `// TODO: TDD RED - Pendiente de implementación` se quita al quedar en verde.
2. Suite completa: `go build ./...` y `go test -count=1 ./...`. Un fallo ajeno al cambio se registra por nombre.
3. Documentar el hallazgo con `## Cambio aplicado` (archivos y verificación) y registrarlo con `findings.py` (3.7).
4. Reinicio del MID con el procedimiento de 3.2. Si el tester es dueño de la instancia (3.2), no hace falta confirmación adicional. Antes de detener el proceso, confirmar que es el que escucha en el puerto del MID.
5. Verificación en vivo solo de los casos afectados, no de toda la matriz, salvo que el usuario lo pida. Antes de cualquier escritura, respaldo (3.4) y línea base de conteos. Si un caso escribe y no debía, restaurar el respaldo y verificar conteos.
6. Registrar el resultado de la verificación en vivo en el hallazgo. Si la corrida de la matriz cambia, actualizar `escritura/resultados.md` con la fecha y el MID usado.
7. Sin commit. Las correcciones quedan en el árbol de trabajo hasta que el dueño decida.

### 3.10 Cambios de especificación e identificadores en cero

- Un `Id` o `TerceroId` igual a 0 o null, o negativo, cuenta como ausente. Un PATCH de la raíz necesita un identificador mayor que cero, por `Id` o por `TerceroId`.
- Si el sistema y `openspec/specs/` difieren, el hallazgo se registra con el texto de la especificación citado y una propuesta de texto. La especificación no se modifica desde estas pruebas. El cambio se aprueba y se aplica por separado.
- Cada hallazgo que cambia un contrato incluye en su `finding.md` una sección de notas para programar: conflicto, texto propuesto, riesgos verificados y pendientes.

## 4. Documentos de estado

Este documento no contiene estado. El estado se guarda en estos archivos, y cada uno tiene una sola función:

| Archivo | Contenido | Cuándo se actualiza |
|---|---|---|
| `manual_tests.md`, `utils/README.md`, `escritura/matriz.md`, `scripts/`, `requests/`, `seed/` | Instrucciones, formato y casos esperados. | Solo cuando cambia el procedimiento. |
| `findings/findings.md`, sección Resumen | Fecha de la última comprobación, conteos, estado de las pruebas Go, último registro de la matriz, servicios y pendientes. | Al cerrar cada sesión (3.8). |
| `findings/findings.md`, sección Índice | Una fila por hallazgo. | Al cambiar un hallazgo, con `findings.py`. |
| `findings/<ID>/finding.md` | Explicación, causa, propuesta, decisión y cambio aplicado de un hallazgo. | Al cambiar ese hallazgo, con `findings.py`. |
| `escritura/resultados.md` | Registro de la última corrida de la matriz: fecha, código HTTP por caso y discrepancias. | Al terminar cada corrida de la matriz. |
| `responses/` | Salida cruda de las corridas. | En cada corrida. |

Reglas:

- Las decisiones pendientes están en `findings/findings.md`, sección Decisiones pendientes.
- Ninguna fecha, conteo o estado de ejecución se escribe en este documento ni en `matriz.md`.
- Un resumen de estado desactualizado se corrige antes de usarlo como punto de partida.

## 5. Reglas

Responsabilidades:

- El tester despliega el MID, garantiza el acceso a su puerto y su logging, y modifica solo el código fuente del MID. Es dueño de la instancia que arranca o reemplaza (3.2): puede darla de baja y recargarla sin confirmación adicional. Una instancia que no arrancó no se da de baja sin preguntar.
- El ambiente despliega `agora_db` (esquema `proveedores`) y el CRUD `gestion_personas` en `8081`.
- Cualquier ajuste al ambiente, a la base o al CRUD se solicita al responsable. No se realiza desde estas pruebas.

Pruebas:

- Lecturas: solo `GET` y `POST /consulta`.
- Escrituras: solo en el esquema `proveedores`, solo desde la matriz, siempre con respaldo previo y restauración al final.
- Autosuficiencia: este documento, `escritura/`, `findings/`, `utils/` y los datos de `seed/` y `snapshots/` bastan para retomar el trabajo. Para el estado actual, leer el Resumen de `findings/findings.md`.
- Ningún cuerpo de prueba incluye el objeto `Tercero`. `TerceroId` se envía como escalar.
- Terceros, parámetros y ubicaciones son de solo lectura desde el MID (F17).
- No se borran datos reales. Las filas que se borran son las que creó una prueba.
- Una afirmación sobre un servicio externo se confirma con un GET de solo lectura antes de publicarse.
- Si la especificación y el sistema difieren, se registra el hallazgo y se decide con el dueño. No se cambia la especificación para que una prueba pase.

Git y código:

- Git: solo comandos de lectura. Sin commits.
- Código: pruebas antes que la implementación, con el marcador `// TODO: TDD RED - Pendiente de implementación` mientras la prueba esté en rojo. Documentación inline en funciones nuevas o modificadas.
- Antes de planear o generar código, leer la skill `test-driven-development`. Antes de tocar código, leer `inline-documentation`. Sin esa lectura no se continúa.

Entorno:

- `.env` y `conf/app.conf` no los lee ni los modifica ningún agente. Los gestiona el ambiente.

Scripts de gestión:

- `utils/findings.py` gestiona solo los hallazgos. Se usa según `utils/README.md`. Para consultar o cambiar un hallazgo se usa el script, no la lectura directa de los archivos.
- Regla de aceptación: ningún hallazgo nuevo o modificado se acepta si `python3 manual_tests/utils/findings.py verificar` reporta incumplimientos.
- El gestor de hallazgos no sustituye `psql`, `curl` ni el resto del procedimiento de pruebas.

## 6. Decisiones pendientes

Están en `findings/findings.md`, sección Decisiones pendientes, ordenadas por prioridad.
