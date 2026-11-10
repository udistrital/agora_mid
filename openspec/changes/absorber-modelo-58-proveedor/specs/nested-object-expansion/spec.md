# Spec Delta

## ADDED Requirements

### Requirement: Proyección aplanada por PK compartida
El sistema SHALL proyectar los campos de una entidad de especialización unida por
PK compartida (por ejemplo `proveedor_natural` o `proveedor_juridico`, cuyo `Id`
es el del proveedor) **dentro** del objeto padre, en lugar de anidarlos bajo un
`mapped_name`, cuando la arista declara la proyección aplanada. La proyección es de
un solo salto y en la dirección padre->especialización.

#### Scenario: Los campos de la especialización se aplanan en el padre
- **WHEN** se expande un proveedor cuya especialización (según `tipo_proveedor`) se
  une por PK compartida y su arista declara proyección aplanada
- **THEN** los campos de esa especialización aparecen al nivel del objeto proveedor,
  no bajo un `mapped_name` anidado

#### Scenario: Solo se aplana la especialización indicada por el discriminador
- **WHEN** un proveedor tiene `tipo_proveedor` de persona jurídica
- **THEN** se aplanan los campos de `proveedor_juridico` y no los de
  `proveedor_natural`, y a la inversa para persona natural

#### Scenario: La proyección aplanada no recurre
- **WHEN** se aplana la especialización de un proveedor
- **THEN** la proyección ocurre una sola vez (un salto) y la especialización no
  vuelve a expandir el proveedor, de modo que no se forma un ciclo de expansión

### Requirement: Entidades del modelo extendido del proveedor
El sistema SHALL expandir, como parte del árbol efectivo del proveedor, las
entidades del modelo extendido: `estado_proveedor`, `documento`, `perfil_proveedor`,
`titulo_proveedor_natural`, `actividad_economica_proveedor`, `representacion` y
`persona_juridica_tipo_entidad`, cada una con su cardinalidad declarada y los campos
`readable` de su tabla de campos tipada.

#### Scenario: Colecciones del modelo extendido se expanden por su cardinalidad
- **WHEN** `fields` selecciona una colección del modelo extendido (por ejemplo
  `PerfilProveedor` o `EstadoProveedor`)
- **THEN** la relación se entrega como lista bajo su `mapped_name`, respetando su
  cardinalidad y los campos `readable` del objeto

#### Scenario: Referencia de un salto sin expansión
- **WHEN** una entidad declara una FK a profundidad 0 hacia otra entidad (por
  ejemplo `representacion.representante_id` o `cuenta_bancaria.contacto_id`)
- **THEN** la referencia se entrega como identificador plano y la entidad
  referenciada no se expande
