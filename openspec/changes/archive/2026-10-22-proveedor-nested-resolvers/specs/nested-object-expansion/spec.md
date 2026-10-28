# Spec Delta

## ADDED Requirements

### Requirement: Despacho a resolvers específicos por relación
El sistema SHALL permitir que cada relación declarada se resuelva mediante un resolver específico asociado por convención de nombre e inyectado por la capa de negocio, y SHALL usar el resolver genérico solo como respaldo cuando una relación no tiene resolver específico asociado.

#### Scenario: Relación con resolver específico asociado
- **WHEN** una relación declarada tiene un resolver específico asociado por su nombre
- **THEN** el expansor delega la consulta de esa relación a ese resolver específico y no al genérico

#### Scenario: Relación sin resolver específico cae al genérico
- **WHEN** una relación declarada no tiene resolver específico asociado
- **THEN** el expansor usa el resolver genérico y se agrega un aviso de tipo `generic_resolver`

#### Scenario: El mapeo se inyecta desde la capa de negocio
- **WHEN** la capa de negocio invoca la expansión
- **THEN** provee el mapeo de nombre de relación a resolver específico, y el expansor no conoce ese mapeo por sí mismo

#### Scenario: La configuración no nombra implementaciones de código
- **WHEN** se declara una relación en la configuración
- **THEN** la asociación con su resolver específico ocurre por convención de nombre en el código y la configuración no incluye nombres de implementaciones

### Requirement: Resolvers específicos como adaptadores de consulta
El sistema SHALL implementar los resolvers específicos como envoltorios finos sobre el cliente HTTP que fijan el endpoint y el campo de consulta real de cada entidad relacionada, reutilizando el transporte, los timeouts y la decodificación existentes, sin aplicar lógica de negocio.

#### Scenario: Consulta por el campo real de la entidad
- **WHEN** una entidad relacionada se consulta por un campo distinto de su propio identificador (por ejemplo por la llave del padre o por una llave primaria compartida)
- **THEN** su resolver específico fija ese campo de consulta y recupera los registros correspondientes

#### Scenario: Sin transformación de negocio en el resolver
- **WHEN** un resolver específico recupera los registros de su entidad
- **THEN** los anida tal como los entrega el CRUD, sin marcar, filtrar ni decidir ramas de negocio

#### Scenario: Reutilización del transporte existente
- **WHEN** un resolver específico ejecuta su consulta
- **THEN** reutiliza el cliente HTTP, los timeouts y la decodificación del envoltorio OAS o lista directa ya existentes, sin reimplementarlos
