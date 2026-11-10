# Tasks

## 1. Configuración del árbol #58 y tabla de campos tipada

- [ ] 1.1 Escribir la prueba de carga (con marcador de rojo) que verifica que el árbol efectivo del proveedor compuesto al arranque contiene las entidades del modelo extendido (`estado_proveedor`, `documento`, `perfil_proveedor` 1-N, `titulo_proveedor_natural`, `actividad_economica_proveedor`, `representacion`, `persona_juridica_tipo_entidad`) con su cardinalidad, y confirmar que falla contra la configuración actual
- [ ] 1.2 Escribir la prueba de carga (con marcador de rojo) que verifica que la tabla de campos tipada de `proveedor_natural` (~18) y `proveedor_juridico` (~13) y el campo `proveedor.consentimiento` corresponden al modelo #58, y que `cuenta_bancaria` usa `tipo_cuenta_id` sin `tipo_cuenta_bancaria_id`, y confirmar que falla
- [ ] 1.3 Reescribir `config/relaciones_proveedor.yaml` al árbol #58 en forma de objetos direccionables + referencias y poblar la tabla de campos tipada por objeto, y verificar que las pruebas 1.1–1.2 pasan y se retiran sus marcadores

## 2. Aplanado de la especialización por PK compartida

- [ ] 2.1 Escribir la prueba de expansión (con marcador de rojo) que verifica que al expandir un proveedor sus campos de especialización (según `tipo_proveedor`) aparecen al nivel del proveedor y no bajo un `mapped_name`, y que un listado mixto aplana por registro, y confirmar que falla
- [ ] 2.2 Escribir la prueba de expansión (con marcador de rojo) que verifica que la proyección aplanada es de un salto y no reexpande el proveedor desde la especialización, y confirmar que falla
- [ ] 2.3 Implementar la proyección aplanada por PK compartida en el motor de expansión sobre la arista existente, y verificar que las pruebas 2.1–2.2 pasan y se retiran sus marcadores
- [ ] 2.4 Escribir la prueba de escritura (con marcador de rojo) que verifica que un cuerpo plano con campos de la especialización se enruta por PK compartida al CRUD dueño (inspección de las peticiones a `proveedor` y a `proveedor_juridico`/`proveedor_natural`), y que un campo de especialización ajeno a `tipo_proveedor` produce `400` sin escrituras, y confirmar que falla
- [ ] 2.5 Implementar el enrutado de campos de especialización aplanados por PK compartida en la escritura, y verificar que la prueba 2.4 pasa y se retira su marcador

## 3. Cascada de escritura del árbol extendido

- [ ] 3.1 Escribir la prueba (con marcador de rojo) del registro de persona natural que verifica que el proveedor y sus entidades admitidas del modelo extendido se crean enviando cada campo a su CRUD dueño, con las FK propagadas (inspección de peticiones), y confirmar que falla
- [ ] 3.2 Escribir la prueba (con marcador de rojo) del registro de persona jurídica, incluida la rama `persona_juridica_tipo_entidad` bajo `proveedor_juridico`, y confirmar que falla
- [ ] 3.3 Escribir la prueba (con marcador de rojo) de que `representacion.RepresentanteId` y `cuenta_bancaria.ContactoId` se envían como FK planas a profundidad 0 sin crear ni actualizar la entidad referenciada, y confirmar que falla
- [ ] 3.4 Implementar la cascada de escritura del árbol extendido (resolvers/adaptadores de las entidades nuevas y propagación de llave), y verificar que las pruebas 3.1–3.3 pasan y se retiran sus marcadores

## 4. Autoría por cuerpo y auditoría administrada

- [ ] 4.1 Escribir la prueba (con marcador de rojo) de que `UsuarioModificacionId` enviado en el cuerpo de una entidad con auditoría se propaga al CRUD dueño como autor de la modificación (inspección de la petición), y confirmar que falla
- [ ] 4.2 Escribir la prueba (con marcador de rojo) de que `FechaCreacion`, `FechaModificacion` o `Activo` en el cuerpo producen `400` como campo no escribible sin materializar escrituras, y confirmar que falla
- [ ] 4.3 Implementar la autoría por cuerpo (`UsuarioModificacionId` escribible) retirando el camino de autoría por contexto de petición, y el rechazo de los demás campos de auditoría, y verificar que las pruebas 4.1–4.2 pasan y se retiran sus marcadores

## 5. Transición de estado del proveedor

- [ ] 5.1 Escribir la prueba (con marcador de rojo) de que registrar un `estado_proveedor` con `estado_id` distinto del activo vigente crea el nuevo estado y da de baja el previo (a lo sumo un activo), con el nuevo creado antes que la baja del previo (inspección del orden de peticiones), y confirmar que falla
- [ ] 5.2 Escribir la prueba (con marcador de rojo) de que registrar un `estado_proveedor` con el `estado_id` del activo vigente es un no-op idempotente (ni crea ni da de baja), y confirmar que falla
- [ ] 5.3 Implementar en el servicio de escritura del proveedor la composición de la transición (resolver el activo vigente, marcar el previo para baja cuando el `estado_id` difiere, crear el nuevo antes de la baja) sobre la capacidad genérica de eliminación del motor, y verificar que las pruebas 5.1–5.2 pasan y se retiran sus marcadores

## 6. Documentación del contrato y verificación de integración

- [ ] 6.1 Actualizar en el contrato publicado y `API.md` el nombre del campo de auditoría de autor a `UsuarioModificacionId`, y verificar que el único requisito de `openapi-publication` que lo nombra queda coherente con la documentación generada
- [ ] 6.2 Ejecutar `go test ./...` y `go build ./...` y verificar que toda la suite pasa sin marcadores de rojo remanentes
