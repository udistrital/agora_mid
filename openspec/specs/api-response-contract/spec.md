# api-response-contract Specification

## Purpose

Define el contrato de respuesta uniforme del API en formato OAS plano más una extensión opcional `MetaData` para avisos de expansión y el total de valores posibles de las listas, alineado con el envoltorio estándar del ecosistema (utils_oas).

## Requirements

### Requirement: Envoltorio de respuesta OAS plano
El sistema SHALL responder con un envoltorio OAS plano que incluya `Success` (booleano), `Status` (cadena con el código HTTP), `Message` (cadena) y `Data` (carga útil), en todos los verbos del API.

#### Scenario: Respuesta exitosa de lectura
- **WHEN** una consulta de lectura se resuelve correctamente
- **THEN** la respuesta incluye `Success: true`, `Status: "200"`, un `Message` y el recurso en `Data`

#### Scenario: Respuesta de escritura
- **WHEN** una operación de escritura (`POST` o `PATCH`) se completa sin fallos parciales
- **THEN** la respuesta usa el mismo envoltorio plano `Success/Status/Message/Data` sin el bloque `MetaData`

#### Scenario: Respuesta de escritura con materialización parcial
- **WHEN** una operación de escritura tolerante materializa solo parte del subárbol por un fallo de relación no bloqueante
- **THEN** la respuesta usa `Status: "206"`, incluye `MetaData.Errors` con los avisos y `Data` con los identificadores ya materializados

#### Scenario: Compatibilidad del envoltorio con el ecosistema
- **WHEN** el controlador de errores estándar y los controladores de negocio emiten respuestas
- **THEN** ambos usan la misma raíz plana `Success/Status/Message/Data` con `Status` como cadena

### Requirement: Extensión MetaData para avisos y total de listas
El sistema SHALL incluir un bloque `MetaData` opcional que puede llevar una colección opcional `Errors` con los avisos de expansión y un campo opcional `Length` con el total de valores posibles de un `Data` de tipo lista. Cada campo del bloque es opcional y se incluye únicamente cuando aplica; `MetaData` se omite cuando no aporta ningún dato. El bloque `Errors` se incluye solo cuando existe al menos un aviso de expansión (error de relación tolerado, reducción de profundidad, ciclo o uso del resolver genérico).

#### Scenario: Expansión parcial con relación tolerada
- **WHEN** una expansión en modo `notify` sufre un error de relación tolerado
- **THEN** la respuesta usa `Status: "206"` e incluye `MetaData.Errors` con una entrada que lleva `Path`, `Service`, `Message` y `Kind`

#### Scenario: Errors es opcional
- **WHEN** una respuesta incluye `MetaData` por otro motivo (por ejemplo `Length`) pero no genera avisos de expansión
- **THEN** el campo `Errors` se omite del bloque `MetaData`

#### Scenario: Total de lista en Length
- **WHEN** una consulta de lectura devuelve un `Data` de tipo lista
- **THEN** `MetaData.Length` refleja el número total de valores posibles sin considerar `limit` ni `offset`, y no el tamaño de la página devuelta

#### Scenario: Length es opcional
- **WHEN** `Data` no es una lista o no aplica un total
- **THEN** el campo `Length` se omite del bloque `MetaData`

#### Scenario: Respuesta sin MetaData aplicable lo omite
- **WHEN** una respuesta no genera avisos de expansión ni reporta un total de lista
- **THEN** el bloque `MetaData` se omite por completo de forma consistente

#### Scenario: Distinción de tipos de aviso
- **WHEN** se registran avisos de distinto origen
- **THEN** cada entrada de `MetaData.Errors` identifica su `Kind` entre `relation`, `depth`, `cycle` o `generic_resolver`
