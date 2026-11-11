# Proposal

## Why

La configuración y los contratos del proveedor en el MID fueron escritos como
preparación, antes de que aterrizara el modelo #58: el árbol de entidades, las
listas de campos de `proveedor_natural` / `proveedor_juridico` y varias relaciones
son marcadores de posición que no corresponden al modelo real. Este cambio absorbe
el modelo #58 como el árbol efectivo real del proveedor —lectura, escritura y
expansión juntas, por entidad— sobre los objetos direccionables y la tabla de
campos tipada que introduce `config-composicion-objetos-tipados`, y apoyándose en
el control de escritura por relación de `control-escritura-por-relacion`.

## What Changes

- **BREAKING (contrato)** El árbol del proveedor se reescribe para ser fiel a #58.
  Entidades nuevas como objetos referenciados: `estado_proveedor`, `documento`,
  `perfil_proveedor` (1-N, hijo directo de proveedor), `titulo_proveedor_natural`
  (1-N), `actividad_economica_proveedor` (1-N), `representacion` (1-N),
  `persona_juridica_tipo_entidad` (1-N bajo `proveedor_juridico`). Se expanden los
  campos de `proveedor_natural` (~18) y `proveedor_juridico` (~13) a los del modelo,
  y se agrega `proveedor.consentimiento`. El campo raíz discriminante es
  `tipo_proveedor`.
- **Especialización por PK compartida (proyección aplanada de un salto).**
  `proveedor_natural` y `proveedor_juridico` comparten la PK con `proveedor`
  (`proveedor.id = proveedor_{natural,juridico}.id`). Al expandir `proveedor`, los
  campos de la especialización correspondiente según `tipo_proveedor` se **aplanan**
  dentro del objeto proveedor en lugar de anidarse bajo un `mapped_name`. En
  escritura, un cuerpo plano de proveedor que incluye campos de la especialización
  se **enruta por PK compartida** al CRUD dueño (`proveedor_juridico` /
  `proveedor_natural`). La proyección es de un solo salto y solo en la dirección
  proveedor->especialización.
- **Autoría de la modificación por cuerpo.** `UsuarioModificacionId` pasa a ser un
  campo **escribible** por cada entidad con campos de auditoría, y su valor proviene
  del **cuerpo** de la escritura. Esto **supera la especificación técnica 8 de
  issue-65** (que pedía autoría desde el contexto de autenticación). El resto de
  campos de auditoría (`fecha_creacion`, `fecha_modificacion`, `activo`) son
  administrados por el sistema y **se rechazan** si llegan en el cuerpo
  (campo desconocido -> `400`).
- **Transición de estado del proveedor.** Registrar un `estado_proveedor` con un
  `estado_id` distinto del estado activo vigente da de baja (borrado, el CRUD lo
  trata como baja lógica) el estado activo previo y crea el nuevo; registrar el
  mismo `estado_id` ya activo es un no-op idempotente. La baja del estado previo se
  expresa en el árbol que el servicio entrega al motor y se materializa por la
  capacidad genérica de eliminación de `control-escritura-por-relacion`.
- **El MID expande cualquier esquema; escribe solo en `terceros` y `proveedores`.**
  La expansión de lectura puede recorrer entidades de cualquier CRUD (`parametros`,
  `ubicaciones`, `terceros`); la materialización de escritura se restringe a los CRUD
  de `terceros` y `proveedores`. Qué FK se expande y qué FK queda como identificador
  plano lo decide el *sidecar* de la generación, no una regla fija por campo.
- **Renombre del campo de auditoría en la documentación.** El nombre del campo de
  auditoría de autor cambia de `AutorModificacionId` a `UsuarioModificacionId` en el
  único requisito de `openapi-publication` que lo nombra.
- **La configuración se genera desde el DBML.** La configuración de relaciones deja
  de escribirse a mano y se produce con un generador (herramienta de `utils/`,
  ejecución manual) a partir del DBML más cuatro *sidecars* de intención: `expand`
  (qué FK se expanden), `write` (política CRUD por objeto, con `natural_key`
  opcional), `fk_inbound` (aristas de llave entrante, poco frecuentes) y `blacklist`
  (columnas que no llegan a la configuración). La salida es determinista y se valida
  contra el DBML.

## Capabilities

### New Capabilities
- `config-generation`: Un generador determinista que transforma el modelo DBML y un
  conjunto de *sidecars* de intención en los archivos de configuración de
  relaciones. El generador vive fuera del arranque y se ejecuta manualmente.

### Modified Capabilities
- `proveedor-write`: El contrato de escritura absorbe el árbol #58 completo (nuevas
  entidades, campos de especialización aplanados y enrutados por PK compartida,
  `consentimiento`); `UsuarioModificacionId` escribible por cuerpo y auditoría
  restante rechazada; transición de `estado_proveedor` con baja del activo previo. La
  escritura se materializa solo en los CRUD de `terceros` y `proveedores`.
- `proveedor-read`: La expansión cubre el árbol #58 completo y aplana los campos de
  la especialización (`proveedor_natural`/`proveedor_juridico`) dentro del proveedor
  según `tipo_proveedor`, de un salto y solo en la dirección proveedor->
  especialización.
- `nested-object-expansion`: Se añade la regla de proyección aplanada por PK
  compartida (de un salto) y la población de la tabla de campos tipada para las
  entidades nuevas del modelo #58.
- `openapi-publication`: Renombre del campo de auditoría de autor
  `AutorModificacionId` -> `UsuarioModificacionId` en el único requisito que lo
  nombra; ningún otro cambio.

## Impact

- **Dependencia de planeación**: se apila sobre `config-composicion-objetos-tipados`
  (objetos direccionables + tabla de campos tipada), `control-escritura-por-relacion`
  (política por relación, no-reparent y la capacidad genérica de eliminación dirigida
  por el árbol que la transición de estado consume) y
  `config-estructura-directorios-por-api`, que debe estar aplicado: la configuración
  generada se escribe en la disposición por subdirectorio de CRUD.
- **Herramienta** (fase apply): el generador se agrega bajo `utils/` para ejecución
  manual; no forma parte de la preparación de arranque y el MID nunca interpreta DBML
  en tiempo de ejecución.
- **Configuración** (fase apply): la configuración del proveedor se **genera** desde
  el DBML y los sidecars al árbol #58 en la forma de objetos direccionables +
  referencias, sobre la disposición por subdirectorio de CRUD; se puebla la tabla de
  campos tipada por objeto.
- **Código** (fase apply): se retira el camino de autoría desde el contexto de
  petición y se sustituye por el campo de cuerpo `UsuarioModificacionId`; el servicio
  de escritura del proveedor compone la transición de `estado_proveedor` (crear el
  nuevo, marcar para baja el activo previo) sobre el motor genérico.
- **Comportamiento observable**: el contrato de `POST`/`PATCH /v1/proveedor` y la
  expansión de lectura reflejan el árbol #58; respuestas `400` ante campos de
  auditoría administrados por el sistema; ningún endpoint nuevo.
- **Conexión**: se usa `AGORA_MID_GESTION_PERSONAS_CRUD_URL` y las variables de
  terceros y parámetros existentes; sin direcciones ni credenciales en el código.
