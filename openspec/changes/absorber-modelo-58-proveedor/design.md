# Design

## Context

Ver `proposal.md` (Why) y los requisitos en `specs/`. Estado actual relevante: el
árbol del proveedor en `config/relaciones_proveedor.yaml` es un marcador de
posición escrito antes de #58; su discriminante (`TipoRegistroId`), varias
relaciones (`Perfil` 1-1 anidado) y las listas de campos de las especializaciones no
corresponden al modelo extendido. El motor de escritura y el de expansión son
genéricos sobre el árbol; este cambio los alimenta con el árbol #58 sobre los objetos
direccionables y la tabla de campos tipada de `config-composicion-objetos-tipados`, y
se apoya en `control-escritura-por-relacion` (política por relación, no-reparent y la
capacidad genérica de eliminación dirigida por el árbol).

Hoy la autoría se toma del contexto de la petición y se inyecta en cada escritura
como un campo que el cuerpo del cliente no puede fijar. Este cambio invierte esa
fuente.

## Goals / Non-Goals

**Goals:**
- Árbol efectivo del proveedor fiel al modelo extendido, en lectura, escritura y
  expansión, por entidad.
- Especialización (`proveedor_natural`/`proveedor_juridico`) como proyección
  aplanada por PK compartida, de un salto, en la dirección proveedor->
  especialización.
- Autoría por cuerpo (`UsuarioModificacionId`) y rechazo de los demás campos de
  auditoría.
- Transición de `estado_proveedor` con baja del activo previo, expresada por el
  servicio y materializada por la capacidad genérica de eliminación del motor.
- Configuración generada de forma determinista desde el DBML y los sidecars de
  intención, verificable contra el DBML, con la herramienta fuera del arranque.

**Non-Goals:**
- Proyección inversa (expandir desde `proveedor_juridico` y traer los campos del
  proveedor): fuera de alcance.
- Materializar como subárbol de escritura las entidades de CRUD ajenos a `terceros`
  y `proveedores`: la escritura se restringe a esos dos CRUD. Qué FK se expande en
  lectura y qué FK queda como identificador plano lo decide el sidecar `expand` de la
  generación, no una regla fija por campo.
- Definir el sobre de respuesta o el manejo de fallo parcial *durante* la
  materialización (incluida la recuperación de una transición de estado a medio
  camino): pertenece al cambio posterior de contrato de respuesta.
- El mecanismo de la tabla de campos tipada y de objetos direccionables: lo provee
  `config-composicion-objetos-tipados`; aquí solo se puebla.

## Decisions

### Decisión 1: La especialización es una proyección aplanada sobre la arista de PK compartida existente, no una primitiva de herencia
`proveedor_natural`/`proveedor_juridico` ya se unen al proveedor por PK compartida
(`proveedor.id = especializacion.id`). Lo nuevo es una bandera de proyección en esa
arista: al leer, los campos de la especialización se aplanan en el proveedor; al
escribir, un cuerpo plano se enruta por PK compartida al CRUD dueño de la
especialización. La dirección es proveedor->especialización y de un solo salto.

Por qué: no se necesita una primitiva de herencia. La arista de PK compartida ya
expresa "misma identidad, campos en dos CRUD"; añadir la proyección aplanada es la
diferencia mínima para que "un proveedor jurídico es un proveedor" se lea y escriba
como un solo objeto. Limitar a un salto y una dirección evita la doble declaración
(proveedor pide especialización y especialización pide proveedor) que sería un ciclo
de composición.

Alternativa considerada: objeto de especialización anidado bajo un `mapped_name`
(como el marcador actual). Rechazada: contradice "es un proveedor"; obliga al cliente
a conocer la rama y parte el contrato en dos niveles.

Alternativa considerada: proyección bidireccional (también desde la
especialización). Rechazada y marcada como no-goal: introduce una doble arista que el
detector de ciclos de composición trataría como ciclo, y issue-65 solo ejercita
`/v1/proveedor`.

### Decisión 2: La autoría se toma del cuerpo; los demás campos de auditoría se rechazan
`UsuarioModificacionId` es un campo escribible por entidad con auditoría y su valor
viene del cuerpo. `FechaCreacion`, `FechaModificacion` y `Activo` no son escribibles
y se rechazan con `400` si llegan.

Por qué: el método de servicio no entrega un encabezado de autoría al backend, de
modo que el cuerpo es la única fuente viable del autor. Mantener los demás campos de
auditoría como administrados por el sistema evita que el cliente falsee marcas de
tiempo o el estado activo.

### Decisión 3: La transición de estado es decisión del servicio, mecanismo del motor
El servicio de escritura del proveedor resuelve el estado activo vigente y compone el
árbol que entrega al motor marcando el estado previo para baja cuando el `estado_id`
entrante difiere; si coincide, no marca nada (no-op). El motor materializa la baja
por su capacidad genérica de eliminación gateada por `deletable`. La ordenación es
crear el nuevo estado y dar de baja el previo después.

Por qué: la regla de "un solo estado activo" es lógica de negocio del proveedor, pero
el borrado pasa por el mismo motor gateado y pre-validado que el resto de la cascada,
sin un segundo camino de escritura. Crear antes de dar de baja deja, ante un fallo
intermedio, el proveedor con el nuevo estado activo (y quizá el previo aún activo),
situación recuperable por reconciliación, en vez de un proveedor sin estado activo.

Alternativa considerada: dar de baja primero y crear después. Rechazada: un fallo tras
la baja dejaría al proveedor sin estado activo.

### Decisión 4: Identidad por `estado_id` para la idempotencia de la transición
La transición compara el `estado_id` entrante con el del estado activo vigente;
coincidencia -> no-op, diferencia -> crear + dar de baja.

Por qué: `estado_id` es el valor de catálogo que identifica el estado; reenviar la
misma intención no debe acumular filas de historia ni romper el reenvío idempotente
del resto del árbol.

### Decisión 5: La configuración se genera desde el DBML con ocho sidecars de intención, no a mano
Un generador en su propia subcarpeta bajo `utils/` (ejecución manual, fuera del
arranque) produce los archivos de configuración a partir del DBML (estructura:
tablas, columnas, tipos, FK) y ocho sidecars de intención agrupados por capacidad, direccionados como
`esquema.tabla[.columna]`: `expand` (qué FK se vuelven aristas), `write` (política
CRUD por objeto, con `natural_key` opcional), `fk_inbound` (aristas de llave entrante),
`blacklist` (columnas excluidas), `service` (parámetros de servicio por tabla raíz),
`flatten` (proyección aplanada y supresión de la expansión inversa de una unión por
PK compartida), `field_roles` (banderas de rol por campo, con comodín `ANY` en el
esquema o la tabla y la más específica prevaleciendo) y `lookup` (aristas de consulta
de solo lectura).
Las entradas de `expand` y `fk_inbound` admiten una forma de cadena (profundidad por
omisión) o un mapa con `depth`. La emisión es determinista (orden fijo de objetos,
campos y entradas de lista) y cada entrada de sidecar se valida contra el DBML: una
entrada sin correspondencia falla identificándose.

Solo los esquemas con CRUD (`proveedores` como `gestion_personas_crud`, `terceros`,
`parametros` y `ubicaciones`) generan objetos. Antes de escribir, el generador
elimina la salida generada anterior; si la construcción falla, la salida previa queda
intacta.

El generador emite dos clases de archivo, como las que el cargador consume: objetos
direccionables y, por cada entrada del sidecar `service`, un archivo de servicio
`relaciones_<servicio>.yaml`. La raíz de un servicio es decisión del sidecar
`service`: sus aristas se trasladan al archivo de servicio, su política de escritura
(del sidecar `write`) se emite en `write.policy` del archivo de servicio, y el objeto
raíz se emite con su lista de referencias vacía. La `natural_key` de la raíz
permanece en el objeto raíz. Los objetos no raíz conservan sus aristas y la política
de escritura de un objeto no raíz se emite sobre las aristas que lo apuntan.

Las entradas de la generación comparten ubicación con el generador: el DBML y los
ocho sidecars viven en esa misma subcarpeta bajo `utils/`, fuera del árbol `config/`
en el que el generador escribe su salida. Todo lo de generación —herramienta,
modelo e intención— tiene el mismo propósito y vive junto; la salida generada es lo
único que entra a `config/`.

Por qué: la estructura del DBML no expresa la intención (expandir vs. plano, qué se
escribe, qué se excluye, identidad del registro, parámetros de servicio). Poner esa
intención en mapas legibles y versionados, en vez de en heurísticas del generador,
hace la regeneración repetible y la depuración directa. El DBML no declara llaves
únicas, de modo que la `natural_key` no puede derivarse y se declara en `write`. El
cargador expande un servicio desde las referencias del archivo de servicio y no tiene
campo de política en el objeto, por lo que las aristas y la política de la raíz solo
son cargables en el archivo de servicio. El MID nunca interpreta DBML en tiempo de
ejecución: la salida generada es el contrato, y debe pasar las pruebas de carga del
árbol #58 como cualquier configuración.

Alternativa considerada: heurísticas en el generador (por esquema o por nombre de
columna) sin sidecars. Rechazada: las excepciones quedarían ocultas en código y una
regeneración sorpresiva obligaría a depurar el criterio del generador en vez de leer
un mapa.

### Decisión 6: El aplanado de la especialización se declara con una bandera en la arista de PK compartida, y el sidecar `flatten` suprime la arista inversa
La unión por PK compartida `proveedor.id = especializacion.id` produce la arista
padre->hijo `shared_pk`. El sidecar `flatten` direcciona la columna de PK compartida
del lado hijo (por ejemplo `proveedores.proveedor_juridico.id`) y produce dos efectos
en el generador:

1. Suprime la arista inversa hijo->padre, de modo que la especialización no reexpande
   el proveedor y no se forma un ciclo de composición. Si la misma columna aparece en
   `fk_inbound`, el generador falla identificando la entrada.
2. Marca la arista padre->hijo con `flatten: true`. La arista sigue saliendo de
   `expand` (`proveedores.proveedor.id` no lleva directiva y se expande con normalidad);
   la bandera indica al motor de lectura que los campos de la especialización se
   proyectan al nivel del proveedor en lugar de anidarse bajo `mapped_name`, según
   `tipo_proveedor`.

Por qué: `nested-object-expansion` y la Decisión 1 ya establecen que la arista declara
la proyección aplanada. Una supresión sin bandera dejaría la proyección sin declarar
en la configuración, y el motor no tendría de dónde leerla. Declarar la bandera en la
arista y suprimir la inversa en el mismo sidecar mantiene una sola intención por
unión.

Alternativa considerada: solo suprimir la arista inversa y dejar que el motor decida la
proyección desde `tipo_proveedor`. Rechazada: la proyección quedaría implícita en el
motor y no en la configuración generada.

Alternativa considerada: una bandera de proyección en la arista de `expand`.
Rechazada: ensucia el vocabulario de `expand` con una semántica que solo aplica a las
uniones por PK compartida y mezcla dos intenciones en una entrada.

### Decisión 7: Las banderas de rol por campo se declaran en `field_roles`, no en una heurística por nombre
El rol de lectura y escritura de cada campo se toma del sidecar `field_roles`, cuyas
entradas se direccionan como `esquema.tabla.columna` con comodín `ANY` en el esquema o
en la tabla (la columna es siempre explícita); la entrada más específica prevalece. Esto reemplaza la convención por nombre
de columna del generador (que marcaba `id` y los campos de auditoría como no
escribibles) por una declaración legible: `ANY.ANY.activo`, `ANY.ANY.fecha_creacion`,
`ANY.ANY.fecha_modificacion` administrados por el sistema (no legibles ni escribibles),
`ANY.ANY.usuario_modificacion_id` escribible por cuerpo pero no legible, `ANY.ANY.id`
legible y no escribible, y las FK de retrorreferencia de cada hijo no legibles ni
escribibles.

Por qué: es el mismo argumento de la Decisión 5 aplicado a los roles. Una convención
por nombre oculta la excepción en el código del generador; un mapa la hace explícita y
auditable. El comodín evita declarar las mismas reglas de auditoría en cada tabla.

Hallazgo que respalda el rol de las FK de retrorreferencia (verificado leyendo el
motor, no ejecutando sus pruebas): la FK derivada que porta la llave del padre
(`TerceroId` en fk_inbound, `ProveedorId` en fk, `Id` en shared_pk) la gobierna el
motor por su `key_propagation` con independencia de las banderas de rol: `CampoDerivado`
la reconoce, `conciliarCampoDerivado` concilia el reenvío (igual conserva, difiere en
entidad existente es `400`) y `aplanarParametros` la conserva en el `Data` aunque no
esté en `writable_fields`. Por eso una FK de retrorreferencia con `readable: false,
writable: false` se oculta en lectura y aun así sobrevive el reenvío de escritura. La
invariante del cargador `immutable ⇒ writable` (en `config/tabla_campos.go`) impide
declarar `immutable: true` junto a `writable: false`, de modo que la FK ascendente de
la raíz (`proveedor.tercero_id`) no declara `immutable`; su inmutabilidad la hace
cumplir el motor por `conciliarCampoDerivado`.

### Decisión 8: Las FK hacia catálogos y lugares son aristas de consulta del sidecar `lookup`
`expand` pone la arista en el objeto referenciado (padre -> colección) y `fk_inbound`
la pone en el objeto que porta la columna con propagación `fk_inbound`, lo que activa
la materialización previa en escritura. Ninguno expresa "este objeto muestra el
catálogo que referencia, sin escribirlo". `lookup` emite la arista en el objeto que
porta la columna, `local_to_related`, cardinalidad `one`, sin propagación y sin verbos,
aunque `write` declare política para el referenciado. El cargador compone esa arista
sin bloque `write` (solo lectura).

El nombre de una arista de `lookup` o `fk_inbound` es la columna sin `_id` en
PascalCase (`tercero_id` -> `Tercero`, `pais_nacimiento_id` -> `PaisNacimiento`). Una
entrada de mapa puede declarar `mapped_name`. Un nombre igual a un campo del objeto
falla, porque ocultaría el campo plano; las columnas sin sufijo `_id` (`tipo_proveedor`,
`regimen_contributivo`, `lugar_origen`, `ciudad_expedicion`, `tipo_lugar`) exigen
`mapped_name`. El generador rechaza los ciclos de referencias con la regla del
cargador, para que una entrada que cierre un ciclo falle al generar y no al arrancar.

Por qué: una arista de solo lectura no debe depender de que `write` no declare
política para un catálogo, y un sidecar propio evita sobrecargar `expand` o
`fk_inbound` con una dirección o una propagación que no les corresponde.

Alternativa considerada: una bandera `direction` en las entradas de `expand`. Rechazada
por la misma razón que la Decisión 6: mezcla dos intenciones en un sidecar.

## Contenido de las entradas

Esta sección fija el contenido de `utils/generador_configuracion/entradas/` para la
tarea 2.3. El DBML `agora_modelo_extendido.dbml` es la única entrada confiable; los
sidecars actuales son marcadores y se reemplazan por completo. Las claves usan
`snake_case` del DBML. La profundidad de cada arista es la altura de su subárbol: una
arista sin aristas debajo vale 1.

**`service.yaml`**: solo `proveedores.proveedor`, con `service: proveedor`,
`max_depth: 5`, `on_relation_error: notify`, `max_related_ids: 100` y
`filterable_fields` con todas las columnas de primer nivel, auditoría incluida: `id`,
`tipo_proveedor`, `descripcion_portafolio`, `consentimiento`, `tercero_id`, `activo`,
`fecha_creacion`, `fecha_modificacion`, `usuario_modificacion_id`.

**`write.yaml`**:
- `proveedores.proveedor`: creatable, updatable, no deletable, `natural_key: [tercero_id]`.
- `proveedores.proveedor_natural` y `proveedores.proveedor_juridico`: creatable,
  updatable, no deletable.
- `proveedores.contacto`, `cuenta_bancaria`, `estado_proveedor`, `documento`,
  `perfil_proveedor`, `titulo_proveedor_natural`, `actividad_economica_proveedor`,
  `representacion`, `persona_juridica_tipo_entidad`: creatable, updatable, deletable,
  sin `natural_key`. `estado_proveedor` no lleva clave natural porque una revisión
  puede registrar un estado nuevo con el mismo `estado_id`.
- Sin entradas para `terceros`, `parametros` ni `ubicaciones`: la arista del tercero y
  su subárbol quedan sin verbos por ahora.

**`fk_inbound.yaml`**: `proveedores.proveedor.tercero_id` con `depth: 4`
(`mapped_name` derivado: `Tercero`).

**`flatten.yaml`**: `proveedores.proveedor_natural.id` y
`proveedores.proveedor_juridico.id`.

**`expand.yaml`** (colecciones y PK compartida):
- Profundidad 1: `contacto.proveedor_id`, `cuenta_bancaria.proveedor_id`,
  `estado_proveedor.proveedor_id`, `documento.proveedor_id`,
  `perfil_proveedor.proveedor_id`, `titulo_proveedor_natural.proveedor_id`,
  `actividad_economica_proveedor.proveedor_id`, `representacion.proveedor_id`, todas
  bajo `proveedores`.
- Profundidad 2: `proveedores.proveedor_natural.id`, `proveedores.proveedor_juridico.id`
  (sus catálogos y lugares cuelgan de ellas),
  `proveedores.persona_juridica_tipo_entidad.persona_juridica_id`,
  `terceros.info_complementaria_tercero.tercero_id`,
  `terceros.tercero_tipo_tercero.tercero_id`.
- Profundidad 3: `terceros.datos_identificacion.tercero_id`.
- Profundidad 1: `terceros.seguridad_social_tercero.tercero_id`.

**`lookup.yaml`** (catálogos y lugares, solo lectura):
- Hacia `parametros.parametro`, profundidad 1: `proveedor.tipo_proveedor`
  (`mapped_name: TipoProveedorDetalle`), `estado_proveedor.estado_id`,
  `cuenta_bancaria.tipo_cuenta_id`, `cuenta_bancaria.entidad_bancaria_id`,
  `documento.tipo_documento_id`, `contacto.tipo_canal_contacto_id`,
  `titulo_proveedor_natural.nucleo_basico_conocimiento_snies_id`,
  `proveedor_juridico.procedencia_empresa_id`, `proveedor_juridico.tipo_conformacion_id`,
  `proveedor_juridico.regimen_contributivo` (`mapped_name: RegimenContributivoDetalle`),
  `actividad_economica_proveedor.actividad_economica_id`, todas bajo `proveedores`.
- Hacia `ubicaciones.lugar`, profundidad 2: `proveedores.proveedor_natural.pais_nacimiento_id`,
  `proveedores.proveedor_juridico.ciudad_origen_id`,
  `proveedores.proveedor_juridico.pais_origen_id`, `terceros.tercero.lugar_origen`
  (`mapped_name: LugarOrigenDetalle`), `terceros.datos_identificacion.ciudad_expedicion`
  (`mapped_name: CiudadExpedicionDetalle`).
- Hacia `ubicaciones.tipo_lugar`, profundidad 1: `ubicaciones.lugar.tipo_lugar`
  (`mapped_name: TipoLugarDetalle`).
- Hacia catálogos de `terceros`, profundidad 1:
  `proveedores.perfil_proveedor.perfil_id` y `proveedores.titulo_proveedor_natural.perfil_id`
  (`info_complementaria`), `proveedores.persona_juridica_tipo_entidad.tipo_entidad_id`
  (`tipo_tercero`), `terceros.tercero.tipo_contribuyente_id`,
  `terceros.datos_identificacion.tipo_documento_id`,
  `terceros.info_complementaria_tercero.info_complementaria_id`,
  `terceros.tercero_tipo_tercero.tipo_tercero_id`.

**Quedan como identificador plano** (sin entrada): `cuenta_bancaria.contacto_id`,
`representacion.representante_id`, `seguridad_social_tercero.tercero_entidad_id`,
`info_complementaria_tercero.info_complementaria_tercero_padre_id`,
`parametro.tipo_parametro_id`, `parametro.parametro_padre_id`. Expandir
`tercero_entidad_id` hacia `Tercero` cierra el ciclo `Tercero -> SeguridadSocialTercero
-> Tercero`, que el cargador rechaza; se entrega plano hasta que el motor admita
reutilizar un ancestro con una proyección propia. `representante_id` queda plano porque
traería el subárbol completo del tercero; se revisa si se necesita el nombre del
representante.

**`blacklist.yaml`**: vacío. Las entradas actuales eran de prueba; `lugar_origen` y
`documento_soporte` se conservan.

**`field_roles.yaml`**:
- `ANY.ANY.id`: legible, no escribible.
- `ANY.ANY.activo`, `ANY.ANY.fecha_creacion`, `ANY.ANY.fecha_modificacion`: ni legibles
  ni escribibles.
- `ANY.ANY.usuario_modificacion_id`: escribible, no legible.
- FK de retrorreferencia, ni legibles ni escribibles: las `proveedor_id` de `contacto`,
  `cuenta_bancaria`, `documento`, `estado_proveedor`, `perfil_proveedor`,
  `titulo_proveedor_natural`, `actividad_economica_proveedor` y `representacion`;
  `persona_juridica_tipo_entidad.persona_juridica_id`; y las `tercero_id` de
  `datos_identificacion`, `seguridad_social_tercero`, `info_complementaria_tercero` y
  `tercero_tipo_tercero`.
- `proveedores.proveedor.tercero_id`: legible, escribible e inmutable. Debe ser
  escribible porque es la `natural_key` de la raíz, y el cargador exige que la clave
  natural sea escribible; al ser escribible, la inmutabilidad vuelve a ser declarable.
- Catálogos y lugares, solo `id` y `nombre` legibles: `parametros.parametro` agrega
  `descripcion`, `codigo_abreviacion` y `numero_orden` legibles y deja `tipo_parametro_id`
  y `parametro_padre_id` legibles como identificador plano. `ubicaciones.lugar` deja
  `tipo_lugar` legible. En `ubicaciones.tipo_lugar`, `terceros.info_complementaria`,
  `terceros.tipo_tercero`, `terceros.tipo_contribuyente` y `terceros.tipo_documento`, las
  demás columnas (`descripcion`, `codigo_abreviacion`, `numero_orden`,
  `tipo_de_dato`, `grupo_info_complementaria_id`) no son legibles. Como `ANY` no aplica
  a la columna, cada columna se declara por su nombre, usando `ANY.tabla.columna`
  cuando la regla vale para varias tablas.
- En `terceros.seguridad_social_tercero`, `tercero_entidad_id` queda legible como
  identificador plano.

## Risks / Trade-offs

- [`readable: false` no oculta un campo cuando la consulta no envía `fields`] → El motor
  solo recorta la respuesta por la selección de `fields` y por la lista negra global de
  ejecución (`config/blacklist.go`); sin `fields` entrega el registro completo del CRUD,
  auditoría incluida. Para que la auditoría no salga nunca en lectura, se declara en la
  lista negra global de ejecución, que no genera este generador. Reportado; no se cambia
  el motor en este cambio.
- [Un cuerpo de escritura con un objeto de `lookup` se rechaza] → La arista de consulta
  no tiene bloque `write`, así que el motor responde `400` si el cliente envía
  `PaisNacimiento: {...}`; el cliente envía la FK plana (`PaisNacimientoId`). El reenvío
  del `Data` de escritura no se afecta, porque ese `Data` no incluye expansiones de
  lectura. Reportado.
- [Las aristas de la especialización aplanada] → Los catálogos de `proveedor_natural` y
  `proveedor_juridico` cuelgan de su objeto; el aplanado del motor (tarea 3.3) debe
  conservar esas aristas anidadas al proyectar los campos en el proveedor.

- [El cliente puede afirmar el autor de la modificación] → Aceptado por decisión del
  proyecto: el método de servicio no permite que un encabezado de autoría llegue al
  backend, por lo que el cuerpo es la única fuente; los demás campos de auditoría
  siguen administrados por el sistema.
- [La transición de estado combina creación y baja: fallo parcial a medio camino] →
  La ordenación crear-antes-de-dar-de-baja acota el peor caso a "estado previo aún
  activo" (recuperable), no a "sin estado activo"; la recuperación y el sobre de
  respuesta del fallo parcial pertenecen al cambio posterior de contrato de respuesta.
- [El aplanado de la especialización podría colisionar con el detector de ciclos de
  composición] → La proyección es de un salto y en una sola dirección; no se declara
  la arista inversa, de modo que no hay ciclo de composición.
- [El árbol reescrito diverge del marcador actual] → Las pruebas de lectura y
  escritura del árbol #58 se escriben primero y fallan contra el marcador actual,
  cubriendo las entidades nuevas, el aplanado de la especialización y el enrutado por
  PK compartida.
- [`cuenta_bancaria.tipo_cuenta_bancaria_id` es un campo fantasma en el diccionario]
  → Se omite del modelo efectivo; la cuenta usa `tipo_cuenta_id`.
- [La expansión de ciudades supera el contrato previo de lectura] → La decisión de
  expandir los lugares de `ubicaciones` con su tipo (de un salto) deja obsoleto el
  escenario "Ciudades sin expansión" del contrato de lectura, revisado en el delta de
  `proveedor-read`. La prueba de integración que fija el comportamiento previo
  (`TestLeerProveedorCiudadesSinExpandir`) debe ajustarse en la fase de aplicación.

## Migration Plan

- Configuración: la configuración del proveedor se **genera** desde el DBML y los
  sidecars al árbol #58 en forma de objetos direccionables + referencias, sobre la
  disposición por subdirectorio de CRUD que aporta `config-estructura-directorios-por-api`;
  se puebla la tabla de campos tipada por objeto. No hay migración de datos.
- Herramienta: el generador se agrega en su propia subcarpeta bajo `utils/`, y el DBML
  y los cinco sidecars se ubican en esa misma subcarpeta como entradas de generación
  versionadas, fuera del árbol `config/`.
- Salida: regenerar elimina la salida generada anterior (archivos YAML con el encabezado
  de generado) antes de escribir la nueva. Los archivos ajenos no se tocan. No hay
  migración de datos.
- Código: se retira el camino de autoría por contexto de petición y se adopta el campo
  de cuerpo `UsuarioModificacionId`; el servicio de escritura compone la transición de
  estado sobre el motor genérico.
- Rollback: revertir la configuración y el servicio restituye el contrato previo; no
  hay estado persistido propio del MID que deshacer.
