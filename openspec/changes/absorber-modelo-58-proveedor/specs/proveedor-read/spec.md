# Spec Delta

## ADDED Requirements

### Requirement: Expansión del árbol extendido del proveedor
El sistema SHALL expandir, a partir del proveedor raíz y según `fields`, las
entidades del modelo extendido: `estado_proveedor`, `documento`,
`perfil_proveedor`, `titulo_proveedor_natural`, `actividad_economica_proveedor`,
`representacion` y `persona_juridica_tipo_entidad`, cada una anidada bajo su
`mapped_name` con su cardinalidad declarada.

#### Scenario: Expansión de colecciones del modelo extendido
- **WHEN** `fields` selecciona rutas de colecciones del modelo extendido (por
  ejemplo `PerfilProveedor`, `Documento` o `EstadoProveedor`)
- **THEN** la respuesta anida cada colección bajo su `mapped_name` como lista,
  respetando su cardinalidad

#### Scenario: Referencias de un salto sin expansión en lectura
- **WHEN** una entidad del árbol referencia otra por una FK a profundidad 0 (por
  ejemplo `representacion.representante_id` o `cuenta_bancaria.contacto_id`)
- **THEN** la referencia se entrega como identificador plano y la entidad
  referenciada no se expande

### Requirement: Aplanado de la especialización en la lectura del proveedor
El sistema SHALL proyectar los campos de la especialización del proveedor
(`proveedor_natural` o `proveedor_juridico`, según `tipo_proveedor`) dentro del
objeto proveedor en lugar de anidarlos bajo un `mapped_name`, de un solo salto y
solo en la dirección proveedor->especialización.

#### Scenario: Los campos de la especialización aparecen en el proveedor
- **WHEN** se expande un proveedor y su `tipo_proveedor` indica persona natural o
  jurídica
- **THEN** los campos de la especialización correspondiente aparecen al nivel del
  objeto proveedor, no bajo un objeto anidado

#### Scenario: Listado mixto aplana la especialización de cada registro
- **WHEN** un listado devuelve en la misma página proveedores de persona natural y
  de persona jurídica
- **THEN** cada registro aplana los campos de su propia especialización según su
  `tipo_proveedor`, sin aplicar a toda la página la rama de un único tipo
