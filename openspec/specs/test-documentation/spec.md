# test-documentation Specification

## Purpose

Documenta de forma legible la suite de pruebas de integración y e2e del subárbol
del proveedor, para que la intención, los actores y los datos de cada prueba sean
verificables sin leer el código.

## Requirements

### Requirement: Documentación de la suite de pruebas del subárbol del proveedor
El sistema SHALL mantener un documento de pruebas en `test/tests.md` que, por cada
prueba de integración y e2e del subárbol del proveedor, describa su propósito, los
actores involucrados, las entradas dadas y las salidas esperadas.

#### Scenario: Cada prueba documenta propósito, actores, entradas y salidas
- **WHEN** se consulta `test/tests.md` para una prueba de la suite
- **THEN** el documento indica qué verifica la prueba, los actores (cliente del
  API, middleware y CRUD dueños simulados), las entradas dadas (verbo, cuerpo
  anidado y estado precargado de los CRUD) y las salidas esperadas (código de
  estado, envoltorio de respuesta e interacciones con los CRUD)

#### Scenario: La documentación cubre los flujos de creación y reconciliación
- **WHEN** la suite incluye los flujos de creación (`POST`), reconciliación
  (`PATCH`) y continuación tras creación parcial
- **THEN** `test/tests.md` documenta cada uno de esos flujos con sus entradas y
  salidas esperadas

#### Scenario: La documentación se mantiene coherente con la suite
- **WHEN** se agrega, elimina o cambia el comportamiento de una prueba de la suite
- **THEN** `test/tests.md` se actualiza para reflejar la prueba vigente, sin
  describir pruebas inexistentes
