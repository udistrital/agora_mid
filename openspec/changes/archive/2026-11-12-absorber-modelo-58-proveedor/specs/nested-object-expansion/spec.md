# Spec Delta

## MODIFIED Requirements

### Requirement: Distinción entre API externa y servicio lógico
El sistema SHALL distinguir, en la configuración de cada relación, entre el campo `related_api` que identifica el servicio externo (la API que resuelve la URL base contra `conf/app.conf`, con valores `terceros`, `parametros`, `gestion_personas_crud` o `ubicaciones`) y el campo `related_service` que nombra el servicio o recurso lógico relacionado del dominio y no resuelve URLs.

#### Scenario: related_api resuelve la URL base
- **WHEN** una relación declara `related_api: terceros`
- **THEN** la URL base de la consulta se resuelve a partir de la API `terceros` configurada en `conf/app.conf`, independientemente del valor de `related_service`

#### Scenario: related_api de ubicaciones resuelve su URL base
- **WHEN** una relación declara `related_api: ubicaciones`
- **THEN** la URL base se resuelve a partir de la API `ubicaciones` configurada en `conf/app.conf` (`ubicaciones_crud_url`), y la carga no la rechaza como API desconocida

#### Scenario: related_api obligatorio y válido
- **WHEN** una relación omite `related_api`, lo deja vacío o declara una API que no existe en la configuración de URLs
- **THEN** el API no inicia y reporta un mensaje que identifica el archivo y la relación con el valor de `related_api` inválido

#### Scenario: related_service no resuelve URLs
- **WHEN** una relación declara un `related_service` como nombre lógico distinto de cualquier API
- **THEN** la validación solo exige que `related_service` no esté vacío y no lo compara contra las URLs externas

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
