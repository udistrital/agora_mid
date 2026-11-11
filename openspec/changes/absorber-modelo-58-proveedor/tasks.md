# Tasks

## 1. Generador de configuración desde el DBML y sidecars

- [ ] 1.1 Escribir la prueba del generador (con marcador de rojo) que verifica que una
  tabla del DBML con columnas produce un objeto en PascalCase con sus campos tipados, y
  que una columna en `blacklist` se omite, y confirmar que falla
- [ ] 1.2 Escribir la prueba (con marcador de rojo) que verifica que una FK en `expand`
  produce una arista expandible y una FK ausente de `expand` produce un identificador
  plano, que `write` fija la política CRUD y la `natural_key` por objeto, y que
  `fk_inbound` marca la arista de llave entrante, y confirmar que falla
- [ ] 1.3 Escribir la prueba (con marcador de rojo) que verifica que la salida es
  determinista (dos ejecuciones producen archivos idénticos byte a byte) y que una
  entrada de sidecar sin correspondencia en el DBML falla identificando la entrada, y
  confirmar que falla
- [ ] 1.4 Implementar el generador en su propia subcarpeta bajo `utils/`, con el DBML y
  los cuatro sidecars ubicados en esa misma subcarpeta como entradas (lectura de DBML y
  sidecars, emisión ordenada, validación de sidecars contra el DBML), de ejecución
  manual y fuera del arranque, y verificar que las pruebas 1.1–1.3 pasan y se retiran
  sus marcadores

## 1b. Archivo de servicio, sidecar de servicio y profundidad por arista

- [ ] 1b.1 Escribir la prueba de carga (con marcador de rojo) del quinto sidecar `service` que verifica que una entrada direccionada por tabla raíz aporta `service`, `max_depth`, `on_relation_error`, `max_related_ids` y `filterable_fields`, y que una clave que no es tabla del DBML o un `filterable_fields` que no es columna de la raíz falla identificando la entrada, y confirmar que falla
- [ ] 1b.2 Escribir la prueba de carga (con marcador de rojo) que verifica que una entrada de cadena en `expand` o `fk_inbound` produce la profundidad por omisión y una entrada de mapa con `depth` produce esa profundidad en la arista, que una entrada de mapa mal formada falla identificándose, y que las entradas de cadena existentes siguen cargando, y confirmar que falla
- [ ] 1b.3 Implementar el lector y validador del sidecar `service` contra el DBML y el formato de entrada de cadena-o-mapa con `depth` en `expand` y `fk_inbound`, y verificar que las pruebas 1b.1–1b.2 pasan y se retiran sus marcadores
- [ ] 1b.4 Escribir la prueba de construcción (con marcador de rojo) que verifica que una arista cuyo padre es la raíz de un servicio aparece en las referencias del archivo de servicio y no en el objeto raíz, que el objeto raíz se construye con referencias vacía, que la política de `write` de la raíz se coloca en `write.policy` del servicio (no en el objeto ni en una arista) mientras su `natural_key` permanece en el objeto raíz, y que una arista de un objeto no raíz permanece en su objeto, y confirmar que falla
- [ ] 1b.5 Implementar la partición de aristas por raíz y la colocación de la política de la raíz en la estructura del servicio, y verificar que la prueba 1b.4 pasa y se retira su marcador
- [ ] 1b.6 Escribir la prueba de escritura (con marcador de rojo) que verifica que se emite `relaciones_<servicio>.yaml` en el subdirectorio del CRUD de la raíz, con `service`, `raiz`, `write.policy`, los cuatro parámetros de servicio y las referencias de la raíz, idéntico byte a byte entre dos ejecuciones, y confirmar que falla
- [ ] 1b.7 Implementar el escritor del archivo de servicio integrándolo en la ejecución del generador, añadir el sidecar `service` como entrada versionada junto al DBML con comentarios de formato, actualizar el README del generador con el quinto sidecar y la profundidad por arista, y verificar que la prueba 1b.6 pasa, que el generador se ejecuta a mano con la entrada nueva y produce el objeto raíz con referencias vacía, y se retiran los marcadores

## 1c. Mapeo de ubicaciones, limpieza de salida y precedencia de fk_inbound

- [ ] 1c.1 Escribir la prueba (con marcador de rojo) que verifica que una tabla del esquema `ubicaciones` produce un objeto con `related_api` igual a `ubicaciones`, y confirmar que falla
- [ ] 1c.2 Escribir la prueba (con marcador de rojo) que verifica que al regenerar se eliminan los YAML con encabezado generado que ya no corresponden, se conservan los archivos sin encabezado, y confirmar que falla
- [ ] 1c.3 Implementar el mapeo de `ubicaciones` al CRUD homónimo y la limpieza de la salida generada antes de escribir, y verificar que las pruebas 1c.1–1c.2 pasan y se retiran sus marcadores
- [ ] 1c.4 Escribir la prueba que fija la precedencia de `fk_inbound` sobre `expand` cuando una FK aparece en ambos, y verificar que pasa; la decisión ya la cumplía el código, por lo que esta prueba no fue roja antes de implementar
- [ ] 1c.5 Regenerar `utils/generador_configuracion/salida` dos veces con el comando del generador y verificar que el conjunto de archivos y los bytes coinciden, que `objetos_proveedor.yaml` tiene `referencias: []` y que existe `relaciones_proveedor.yaml`
- [ ] 1c.6 Escribir la prueba que verifica que una entrada de `service` sin `service` falla identificándose, y que una raíz de un esquema sin CRUD falla identificándose, y verificar que pasa contra la implementación actual
- [ ] 1c.7 Documentar en el README del generador el requisito de esquemas con CRUD y la limpieza de la salida, y verificar que el README coincide con el comportamiento de `Generar`

## 1d. Configuración del CRUD de ubicaciones en el cargador

- [ ] 1d.1 Declarar `ubicaciones_crud_url` en `conf/app.conf` y
  `AGORA_MID_UBICACIONES_CRUD_URL` en `.env.example` como la URL base del CRUD de
  ubicaciones
- [ ] 1d.2 Documentar `AGORA_MID_UBICACIONES_CRUD_URL` en la lista de variables de
  entorno del `README.md` y añadir `ubicaciones` a los valores de `related_api` en
  `config/README.md`
- [ ] 1d.3 Escribir la prueba (con marcador de rojo) que verifica que el cargador
  resuelve un objeto con `related_api: ubicaciones` contra `ubicaciones_crud_url` y
  que un árbol con esa API carga sin rechazarse como API desconocida, y confirmar que
  falla contra la configuración actual
- [ ] 1d.4 Añadir `UbicacionesCRUDURL` a `config.Config`, leer `ubicaciones_crud_url`
  en `config.Load` y mapear `ubicaciones` en `Config.URLs`, y verificar que la prueba
  1d.3 pasa y se retira su marcador

## 2. Configuración del árbol #58 y tabla de campos tipada

- [ ] 2.1 Escribir la prueba de carga (con marcador de rojo) que verifica que el árbol efectivo del proveedor compuesto al arranque contiene las entidades del modelo extendido (`estado_proveedor`, `documento`, `perfil_proveedor` 1-N, `titulo_proveedor_natural`, `actividad_economica_proveedor`, `representacion`, `persona_juridica_tipo_entidad`) con su cardinalidad, y confirmar que falla contra la configuración actual
- [ ] 2.2 Escribir la prueba de carga (con marcador de rojo) que verifica que la tabla de campos tipada de `proveedor_natural` (~18) y `proveedor_juridico` (~13) y el campo `proveedor.consentimiento` corresponden al modelo #58, y que `cuenta_bancaria` usa `tipo_cuenta_id` sin `tipo_cuenta_bancaria_id`, y confirmar que falla
- [ ] 2.3 Generar la configuración del proveedor con el generador desde el DBML y los sidecars, poblando el árbol #58 en forma de objetos direccionables + referencias sobre la disposición por subdirectorio de CRUD y la tabla de campos tipada por objeto, y verificar que las pruebas 2.1–2.2 pasan y se retiran sus marcadores

## 3. Aplanado de la especialización por PK compartida

- [ ] 3.1 Escribir la prueba de expansión (con marcador de rojo) que verifica que al expandir un proveedor sus campos de especialización (según `tipo_proveedor`) aparecen al nivel del proveedor y no bajo un `mapped_name`, y que un listado mixto aplana por registro, y confirmar que falla
- [ ] 3.2 Escribir la prueba de expansión (con marcador de rojo) que verifica que la proyección aplanada es de un salto y no reexpande el proveedor desde la especialización, y confirmar que falla
- [ ] 3.3 Implementar la proyección aplanada por PK compartida en el motor de expansión sobre la arista existente, y verificar que las pruebas 3.1–3.2 pasan y se retiran sus marcadores
- [ ] 3.4 Escribir la prueba de escritura (con marcador de rojo) que verifica que un cuerpo plano con campos de la especialización se enruta por PK compartida al CRUD dueño (inspección de las peticiones a `proveedor` y a `proveedor_juridico`/`proveedor_natural`), y que un campo de especialización ajeno a `tipo_proveedor` produce `400` sin escrituras, y confirmar que falla
- [ ] 3.5 Implementar el enrutado de campos de especialización aplanados por PK compartida en la escritura, y verificar que la prueba 3.4 pasa y se retira su marcador

## 4. Cascada de escritura del árbol extendido

- [ ] 4.1 Escribir la prueba (con marcador de rojo) del registro de persona natural que verifica que el proveedor y sus entidades admitidas del modelo extendido se crean enviando cada campo a su CRUD dueño, con las FK propagadas (inspección de peticiones), y confirmar que falla
- [ ] 4.2 Escribir la prueba (con marcador de rojo) del registro de persona jurídica, incluida la rama `persona_juridica_tipo_entidad` bajo `proveedor_juridico`, y confirmar que falla
- [ ] 4.3 Escribir la prueba (con marcador de rojo) de que `representacion.RepresentanteId` y `cuenta_bancaria.ContactoId` se envían como FK planas a profundidad 0 sin crear ni actualizar la entidad referenciada, y confirmar que falla
- [ ] 4.4 Implementar la cascada de escritura del árbol extendido (resolvers/adaptadores de las entidades nuevas y propagación de llave), y verificar que las pruebas 4.1–4.3 pasan y se retiran sus marcadores

## 5. Autoría por cuerpo y auditoría administrada

- [ ] 5.1 Escribir la prueba (con marcador de rojo) de que `UsuarioModificacionId` enviado en el cuerpo de una entidad con auditoría se propaga al CRUD dueño como autor de la modificación (inspección de la petición), y confirmar que falla
- [ ] 5.2 Escribir la prueba (con marcador de rojo) de que `FechaCreacion`, `FechaModificacion` o `Activo` en el cuerpo producen `400` como campo no escribible sin materializar escrituras, y confirmar que falla
- [ ] 5.3 Implementar la autoría por cuerpo (`UsuarioModificacionId` escribible) retirando el camino de autoría por contexto de petición, y el rechazo de los demás campos de auditoría, y verificar que las pruebas 5.1–5.2 pasan y se retiran sus marcadores

## 6. Transición de estado del proveedor

- [ ] 6.1 Escribir la prueba (con marcador de rojo) de que registrar un `estado_proveedor` con `estado_id` distinto del activo vigente crea el nuevo estado y da de baja el previo (a lo sumo un activo), con el nuevo creado antes que la baja del previo (inspección del orden de peticiones), y confirmar que falla
- [ ] 6.2 Escribir la prueba (con marcador de rojo) de que registrar un `estado_proveedor` con el `estado_id` del activo vigente es un no-op idempotente (ni crea ni da de baja), y confirmar que falla
- [ ] 6.3 Implementar en el servicio de escritura del proveedor la composición de la transición (resolver el activo vigente, marcar el previo para baja cuando el `estado_id` difiere, crear el nuevo antes de la baja) sobre la capacidad genérica de eliminación del motor, y verificar que las pruebas 6.1–6.2 pasan y se retiran sus marcadores

## 7. Documentación del contrato y verificación de integración

- [ ] 7.1 Actualizar en el contrato publicado y `API.md` el nombre del campo de auditoría de autor a `UsuarioModificacionId`, y verificar que el único requisito de `openapi-publication` que lo nombra queda coherente con la documentación generada
- [ ] 7.2 Ejecutar `go test ./...` y `go build ./...` y verificar que toda la suite pasa sin marcadores de rojo remanentes
