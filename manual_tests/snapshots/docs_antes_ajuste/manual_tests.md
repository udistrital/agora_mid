# Pruebas manuales de integración de agora_mid

Estado vigente al 14/11/2026. Este documento describe cómo repetir las pruebas, qué está verificado y qué queda pendiente. Los hallazgos están en `findings/findings.md`. Las propuestas para decidir están al final.

## 1. Alcance

- Caja negra: `curl` contra la API del middleware (MID) y contra el CRUD `gestion_personas` (lectura de comprobación).
- Servicios:
  - MID `agora_mid`: `http://localhost:8080` (o el puerto de `AGORA_MID_HTTP_PORT`).
  - CRUD `gestion_personas_crud`: `http://localhost:8081/v1`. Su `/swagger/swagger.json` es un stub y no se usa como contrato (F05).
  - Base: contenedor `agora_db` (PostgreSQL 18), esquema `proveedores`.
  - Dependencias remotas de solo lectura: terceros (`pruebasapi.intranetoas.udistrital.edu.co:8121`) y ubicaciones (`:8085`, no disponible, F15).
- Escrituras: solo en el esquema `proveedores`, desde la matriz de escritura.

### Responsabilidades

| Componente | Responsable | Qué debe garantizar |
|---|---|---|
| Gestión del MID (arranque, parada, estado, logs y puerto HTTP) | Tester | Que el MID esté desplegado, que su puerto sea accesible desde el entorno de pruebas y que sus logs se puedan leer. Antes de usarlo o arrancarlo, verifica si ya está en ejecución. Si lo está, pregunta si debe reiniciarse o ejecutarse en otro puerto. No lo reinicia sin confirmación. |
| Variables de entorno del MID (`.env`) y `conf/app.conf` | Ambiente | Las carga el ambiente en el proceso del MID. Ningún agente ni el tester las leen ni las modifican. |
| Código fuente del MID | Tester | Es el único componente que el tester modifica. |
| Base `agora_db` (esquema `proveedores`) | Ambiente | Despliegue y disponibilidad en `agora_db`, puerto `5432`. |
| CRUD `gestion_personas` | Ambiente | Despliegue y disponibilidad en `8081`, con su propio `.env`. |
| Terceros, ubicaciones, parámetros | Externos al proyecto | Solo lectura. Su disponibilidad no depende del tester. |

Cualquier cambio en la base, en el CRUD o en el ambiente se solicita al responsable. No se realiza desde estas pruebas.

## 2. Estructura

```
manual_tests/
├── manual_tests.md        este documento
├── escritura/
│   ├── matriz.md          casos de escritura (POST y PATCH) y efecto esperado
│   └── resultados.md      estado vigente de la matriz
├── requests/              cuerpos JSON de los casos (escritura/ incluye la matriz)
├── responses/
│   ├── current/           salida de scripts/*.sh (lecturas). Se crea al ejecutar esos scripts.
│   └── escritura/current/ última corrida de la matriz (.json y .status)
├── scripts/               una petición por archivo (lecturas en scripts/, matriz en scripts/escritura/)
├── seed/                  dummy_data.sql (carga) y cleanup_dummy_data.sql (sin probar)
├── snapshots/             respaldo de esquema proveedores para restaurar
├── findings/              findings.md (índice) y una carpeta por hallazgo
└── utils/                 findings.py (gestión de hallazgos). Guía en utils/README.md.
```

## 3. Procedimiento repetible

### 3.0 Lectura previa obligatoria

Antes de ejecutar cualquier paso, el agente o tester lee:

- Este documento completo.
- `utils/README.md`: uso de `findings.py`, formato de los hallazgos, vocabulario y regla de aceptación.
- Las skills `test-driven-development` e `inline-documentation` en `.kiro/skills/`, si va a tocar código.

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

Responsable: tester. Compilar y arrancar el binario. Evita `bee run`, que no termina cuando el binario sale (F11).

Antes de arrancar, comprobar si el MID ya corre:

```shell
pgrep -fa agora_mid_run | grep -v pgrep || echo "sin proceso agora_mid_run"
curl -s -m 3 -o /dev/null -w '%{http_code}\n' http://localhost:8080/v1/proveedor/consulta   # 405 si está arriba
```

Si el MID ya corre, preguntar antes de actuar: reiniciarlo, o ejecutar la nueva instancia en otro puerto (`AGORA_MID_HTTP_PORT`, ver abajo). No matar el proceso sin confirmación.

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

Las respuestas quedan en `responses/current/` (`.json` y `.status`). El directorio se crea al ejecutar los scripts. Son lecturas: no escriben.

### 3.4 Matriz de escritura

Antes de escribir, respaldar. El respaldo debe llevar `--clean --if-exists` para poder restaurarse sobre la base viva.

```shell
cd manual_tests
docker exec agora_db pg_dump -U postgres -d postgres -n proveedores --clean --if-exists > snapshots/proveedores_antes_matriz.sql
```

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

Comparar con `escritura/matriz.md` (código HTTP y efecto). Actualizar `escritura/resultados.md`.

Restaurar la base y verificar conteos. El estado inicial es proveedor 2, natural 1, jurídico 1, contacto 1, cuenta 1.

```shell
docker exec -i agora_db psql -U postgres -d postgres -v ON_ERROR_STOP=1 -q < snapshots/proveedores_antes_matriz.sql
for t in proveedor proveedor_natural proveedor_juridico contacto cuenta_bancaria; do
  printf "%s=" $t; docker exec agora_db psql -U postgres -At -c "select count(*) from proveedores.$t;"
done
```

### 3.5 Pruebas Go

```shell
cd /home/ndsabogalv/go/src/github.com/udistrital/agora_mid
go build ./...
go test -count=1 ./...
```

El único fallo esperado es `TestIntegracionConsultaSinFieldsNoEntregaAuditoria` (F04).

### 3.7 Gestión de hallazgos

Usar siempre `utils/findings.py` según `utils/README.md`:

1. Antes de registrar o cambiar un hallazgo, consultarlo con `ver` o `listar`. Para una sección, `segmento`.
2. Registrar o cambiar con `cambiar <ID> --estado ... --solucion ...`. Primero sin `--aplicar` para revisar la propuesta, luego con `--aplicar`.
3. Después de cada cambio, ejecutar `verificar`. Debe mostrar `incumplimientos: 0`.
4. Ejecutar las pruebas del script: `python3 -W error -m unittest discover -s manual_tests/utils/tests`.

Ningún hallazgo nuevo o modificado se acepta con incumplimientos.

### 3.6 Scripts de carga y lectura de datos

- Carga: `docker exec -i agora_db psql -U postgres -v ON_ERROR_STOP=1 -f - < seed/dummy_data.sql`. Aborta si ya hay filas en las tablas del esquema.
- Limpieza: `seed/cleanup_dummy_data.sql` borra los ids 1 y 2 de los datos dummy. No está probado (F13). Probarlo antes de usarlo.

## 4. Estado vigente

Estado al 14/11/2026, 12:34 (hora local). Repetir las comprobaciones de 3.1 antes de usar estos datos:

- MID: arriba en `8080`, proceso `/tmp/agora_mid_run` (PID 69726), log en `/tmp/agora_mid_run.log`. Arrancado con las variables cargadas desde `.env` mediante el comando de 3.2. `GET /v1/proveedor/consulta` responde 405, como se espera.
- Proceso `bee run` activo (PID 41244): ejecuta `gestion_personas_crud`, no el MID.
- CRUD `8081`: responde 200 a `GET /v1/proveedor/1`.
- Base: conteos iniciales correctos (proveedor 2, natural 1, jurídico 1, contacto 1, cuenta 1; ids de proveedor 1 y 2).
- Lecturas: no hay corrida registrada en `responses/current/`.
- Matriz (corrida del 14/11): 39 casos. Entre los 36 con resultado esperado definido, 34 coinciden en código HTTP. Discrepancias: P11 (F22) y U03 (F23). Sin resultado esperado definido: P04 (F24), U13 (F28, regresión de F19) y U16 (F27). Ver `escritura/resultados.md`.
- Efecto en BD: solo verificado el conteo final. P03 reutiliza el proveedor 1 por clave natural (`natural_key: [TerceroId]`), como indica la especificación. La matriz se corrigió. U09 no tiene efecto verificado.
- F19 corregido en el árbol de trabajo. Los PATCH que daban 502 ahora dan 200 (U01, U11, U18, U19).
- Cuenta bancaria: el CRUD responde 200 y el MID muestra `CuentaBancaria`. U17 pasa a 200.
- Pruebas Go: `go build ./...` pasa. `go test` pasa salvo F04.
- Cambios de código sin commit en el árbol de trabajo. Git solo se usa en lectura.

### Pendientes

- Lecturas (3.3): no se han ejecutado sobre el MID actual. `responses/current/` no existe.
- Matriz (3.4): la última corrida es anterior a la migración de hallazgos. Repetirla requiere respaldo, corrida y restauración según 3.4.
- Efecto por caso: solo P03 (corregido en la matriz) y el conteo final tienen verificación. Los demás efectos de la columna "Efecto en BD" de `escritura/matriz.md` no se han comprobado uno por uno. U09 queda sin verificar.
- Hallazgos: los 24 cumplen el formato de `utils/README.md`. Cada uno tiene estado y solución. Las decisiones de la sección 6 siguen abiertas.
- Variables de entorno: F09, F15 y F24 usan nombres de variable (`AGORA_MID_*`) en lugar de hosts. El procedimiento usa las mismas variables.

## 5. Reglas

Responsabilidades:

- El tester despliega el MID, garantiza el acceso a su puerto y su logging, y modifica solo el código fuente del MID.
- El ambiente despliega `agora_db` (esquema `proveedores`) y el CRUD `gestion_personas` en `8081`.
- Cualquier ajuste al ambiente, a la base o al CRUD se solicita al responsable. No se realiza desde estas pruebas.

Pruebas:

- Lecturas: solo `GET` y `POST /consulta`.
- Escrituras: solo en el esquema `proveedores`, solo desde la matriz, siempre con respaldo previo y restauración al final.
- Autosuficiencia: este documento, `escritura/`, `findings/`, `utils/` y los datos de `seed/` y `snapshots/` bastan para retomar el trabajo. No depende de conversaciones previas.
- Ningún cuerpo de prueba incluye el objeto `Tercero`. `TerceroId` se envía como escalar.
- Terceros, parámetros y ubicaciones son de solo lectura desde el MID (F17).
- No se borran datos reales. Las filas que se borran son las que creó una prueba.
- Una afirmación sobre un servicio externo se confirma con un GET de solo lectura antes de publicarse.
- Si la especificación y el sistema difieren, se registra el hallazgo y se decide con el dueño. No se cambia la especificación para que una prueba pase.

Git y código:

- Git: solo comandos de lectura. Sin commits.
- Código: pruebas antes que la implementación, con el marcador `// TODO: TDD RED - Pendiente de implementación` mientras la prueba esté en rojo. Documentación inline en funciones nuevas o modificadas.
- Antes de planear o generar código, leer las skills `test-driven-development` e `inline-documentation` en `.kiro/skills/`. Sin esa lectura no se continúa.

Entorno:

- `.env` y `conf/app.conf` no los lee ni los modifica ningún agente. Los gestiona el ambiente.

Scripts de gestión:

- `utils/findings.py` gestiona solo los hallazgos. Se usa según `utils/README.md`. Para consultar o cambiar un hallazgo se usa el script, no la lectura directa de los archivos.
- Regla de aceptación: ningún hallazgo nuevo o modificado se acepta si `python3 utils/findings.py verificar` reporta incumplimientos. El caso no debe darse.
- El gestor de hallazgos no sustituye `psql`, `curl` ni el resto del procedimiento de pruebas.

## 6. Propuestas

Ordenadas por prioridad. Ninguna está aplicada salvo que se indique.

1. **F28 (prioridad alta, regresión de F19).** Responder 404 (o 400) en la raíz cuando el `Id` del cliente no existe, sin escribir. Los hijos con PK compartida siguen creándose con el Id del padre (P06 y P19 lo necesitan). Requiere que el motor distinga ambos casos. Decisión pendiente.
2. **F22 (alta).** Validar en el MID los campos requeridos de cada objeto antes de escribir. Devolver 400 con el nombre del campo. Afecta P11.
3. **F23 (alta).** En la raíz, comparar `TerceroId` del cuerpo con el del registro existente y responder 400 si difieren. Afecta U03.
4. **F24 (alta, decisión).** Consultar `GET /v1/tercero/{TerceroId}` al escribir la raíz y devolver 400 si responde 404. Requiere decidir si el MID consulta terceros en cada escritura. Afecta P04.
5. **F17 (decisión).** Elegir entre guarda total de `terceros` (y dejar pendiente la escritura de `tercero`) o una lista explícita de destinos. La segunda requiere cambiar `openspec/specs/proveedor-write`.
6. **F04 (decisión).** Opción A: entrada concreta por recurso en `global_blacklist.yaml`, ajustando el validador. Opción B: validador que consulte campos de objetos generados (cambia la especificación `nested-object-expansion`). Corrige la única prueba roja.
7. **F27 (media).** Responder 400 a un PATCH sin `Id` ni clave natural. Decidir con el dueño del contrato.
8. **F20 (media).** Convertir `TerceroId` 0 a NULL en el CRUD (`TerceroId *int`), o no enviarlo desde el MID cuando es 0.
9. **F14 (decisión).** Dejar de emitir `depth` cuando la relación se resuelve completa, o documentar que el 206 no implica pérdida de datos.
10. **F09 (media).** Comprobar el host de terceros antes de arrancar, o cachear el catálogo `P_NATURAL`.
11. **F07 y F06 (CRUD).** Responder 404 solo para `ErrNoRows`, 500 para el resto. Lista vacía: 200 con `Data: []`. Revalidar F06 primero.
12. **F05, F08, F18 (documentación).** Regenerar el swagger del CRUD. Corregir los nombres de `API.md` según la tabla de naming de `escritura/matriz.md`.
13. **F13 (entorno).** Probar `seed/cleanup_dummy_data.sql` en una base de pruebas y verificar los catálogos del seed.
14. **F10, F11 (operación).** Borrar los YAML sin referencia o ignorar la advertencia. Arrancar siempre el binario, no `bee run`.
15. **F15 (dependencia).** No se corrige desde el MID. Repetir la comprobación del host de ubicaciones antes de cada corrida.
16. **Cambios de atajo.** F03, F25 y F26 están como arreglos rápidos sin commit. Decidir si se reemplazan por un diseño definitivo o se aceptan. F25 y F26 son parches de forma; F03 aplana cualquier mapa con `Id`.
