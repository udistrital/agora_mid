# Proposal

## Why

La infraestructura de expansión de objetos anidados ya existe y está probada
(helper, resolver genérico, carga de configuración de relaciones y contrato de
respuesta OAS plano), pero nunca se integró en un endpoint de lectura: la
integración previa se intentó mediante composición manual específica por persona
(`componerProveedor`, DTOs compuestos con `UnmarshalJSON`/`MarshalJSON` a medida),
que quedó desalineada con el modelo declarativo y debe descartarse. Necesitamos
un endpoint de lectura real centrado en la tabla `proveedor` que entregue sus
entidades relacionadas como objetos anidados usando la configuración declarativa
y resolvers específicos por entidad, sin que el MID se vuelva dueño de los datos.

## What Changes

- Nuevo endpoint de lectura `GET /v1/proveedor?id=&fields=` como única raíz de la
  fase 1. Se consulta exclusivamente por `id` de proveedor; el tercero se alcanza
  anidado (por ejemplo `fields=Tercero.Id,Tercero.NombreCompleto`), no como punto
  de entrada propio.
- Nuevo `ProveedorService` (capa de negocio) que trae el proveedor base por `id`
  desde `gestion_personas_crud`, arma `ExpandOptions` a partir de `fields`,
  registra el mapeo relación→resolver e invoca el helper. Es el único que
  contiene lógica de negocio (marcar datos, decidir rama natural/jurídico,
  conservar el objeto para una posible modificación).
- Resolvers específicos por entidad del núcleo, como envoltorios finos sobre
  `HTTPResolver`: fijan endpoint y campo de consulta real de cada tabla
  (`id` propio, `tercero_id`, `proveedor_id`, o PK compartida) y anidan el
  payload sin lógica de negocio.
- El helper gana **dispatch de relación a resolver por convención de nombre con
  fallback al resolver genérico**: el `ProveedorService` inyecta el registro
  relación→resolver; una relación sin resolver específico cae al genérico.
- Nuevo `config/relaciones_proveedor.yaml` que declara el árbol del núcleo:
  `tercero` (con `datos_identificacion`, `tipo_contribuyente`),
  `proveedor_natural`/`proveedor_juridico` (con `perfil`), `contacto` y
  `cuenta_bancaria`. Las ciudades (`ubicaciones.*`) quedan como ID plano sin
  expandir.
- **BREAKING**: se eliminan los artefactos de composición manual del contrato de
  personas: `componerProveedor` y el `MarshalJSON` a medida de
  `PersonaIntegradaResultado` en `services/consulta_actualizacion.go`, y los DTOs
  compuestos (`registro_compuesto.go`, `actualizar_compuesto.go`) con sus
  `UnmarshalJSON` que rechazaban el contrato por bloques. El shape de salida deja
  de armarse a mano y pasa a producirlo el helper a partir de la configuración.

## Capabilities

### New Capabilities
- `proveedor-read`: lectura del proveedor como raíz por `id`, con expansión de
  sus entidades relacionadas del núcleo mediante `fields` y configuración
  declarativa, orquestada por un servicio de negocio que delega en resolvers
  específicos por entidad.

### Modified Capabilities
- `nested-object-expansion`: el helper deja de depender únicamente del resolver
  genérico; se introduce el despacho relación→resolver por convención de nombre
  con fallback al genérico, y los resolvers específicos (envoltorios finos sobre
  el cliente HTTP) que entienden el origen real de cada tabla.

## Impact

- **Código nuevo**: `controllers/proveedor.go` (1 controlador), `services/`
  (`ProveedorService` raíz + resolvers específicos por entidad del núcleo),
  registro relación→resolver.
- **Código modificado**: `helper/` gana el dispatch por nombre con fallback;
  `main.go` registra la nueva ruta `/v1/proveedor`.
- **Código eliminado**: `componerProveedor` y `MarshalJSON` de
  `PersonaIntegradaResultado`; DTOs compuestos de registro/actualización y sus
  tests asociados (artefactos del commit 7387a47).
- **Config nueva**: `config/relaciones_proveedor.yaml`, cargada y validada al
  arranque por el loader existente.
- **Endpoints afectados**: alta de `GET /v1/proveedor`. BREAKING para cualquier
  consumidor del contrato compuesto manual previo.
- **Alcance fase 1**: núcleo = proveedor, tercero, datos_identificacion,
  tipo_contribuyente, proveedor_natural/juridico, perfil, contacto,
  cuenta_bancaria. Diferidos a fase 2: perfil_financiero, actividad_economica,
  representacion, informacion_financiera, documento, declaracion, ubicaciones.
- **Pruebas**: unitarias del dispatch del helper, de cada resolver específico y
  del `ProveedorService`; de controlador para `fields`, `400`, `206` y
  serialización de avisos.
