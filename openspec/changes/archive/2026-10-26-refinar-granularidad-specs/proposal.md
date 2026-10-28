# Proposal

## Why

Dos requisitos de especificaciones principales superan los 500 caracteres y
describen varios comportamientos normativos en una sola descripción
(`nested-object-expansion` → "Selección de campos por rutas", 972 chars;
`api-response-contract` → "Extensión MetaData para avisos y total de listas",
741 chars). Una descripción tan densa obliga a cualquier lector —agente o
persona— a descomponerla mentalmente en cada lectura y debilita la trazabilidad
requisito↔escenario, lo que ofusca la atención al implementar o auditar. Partir
cada requisito en unidades atómicas (una conducta por requisito, cada una con
sus escenarios) reduce esa carga cognitiva y deja la validación estricta limpia.

## What Changes

- Se divide el requisito `Selección de campos por rutas` de
  `nested-object-expansion` en cuatro requisitos atómicos: la selección y
  resolución de rutas se conserva (MODIFIED, acortada a una sola conducta) y se
  extraen como requisitos nuevos la materialización de segmentos intermedios, el
  recorte de campos y relaciones no alcanzados, y la precedencia por relación.
- Se divide el requisito `Extensión MetaData para avisos y total de listas` de
  `api-response-contract` en tres requisitos atómicos: el bloque `MetaData` y su
  campo `Length` se conservan (MODIFIED, acortado), y se extraen como requisitos
  nuevos la colección `Errors` de avisos y la obligación de que cada aviso
  declare su `Kind` respaldado por una constante extensible.
- No cambia ningún comportamiento observable del sistema: es un refinamiento de
  la forma de las especificaciones, no de la conducta ya implementada y probada.
  Cada escenario existente se conserva y se reparte al requisito atómico al que
  pertenece; donde un requisito nuevo quede sin escenario propio, se redacta uno
  derivado del texto normativo original sin alterar su significado.

## Capabilities

### New Capabilities
<!-- Ninguna capacidad nueva: solo se reestructuran requisitos de capacidades existentes. -->

### Modified Capabilities
- `nested-object-expansion`: se parte el requisito "Selección de campos por
  rutas" en cuatro requisitos atómicos (uno conservado y acortado, tres nuevos),
  preservando todos sus escenarios y su significado normativo.
- `api-response-contract`: se parte el requisito "Extensión MetaData para avisos
  y total de listas" en tres requisitos atómicos (uno conservado y acortado, dos
  nuevos), preservando todos sus escenarios y su significado normativo.

## Impact

- **Especificaciones**: `openspec/specs/nested-object-expansion/spec.md` y
  `openspec/specs/api-response-contract/spec.md` pasan de 2 requisitos extensos a
  7 requisitos atómicos (todos < 500 caracteres).
- **Validación**: `openspec validate --specs --strict` deja de reportar los dos
  warnings de longitud; la validación normal ya pasaba.
- **Código**: ninguno. El comportamiento ya está implementado y cubierto por
  pruebas; este cambio no toca `controllers/`, `services/`, `helper/`,
  `swagger/` ni `main.go`.
- **Consumidores**: ninguno. Proyecto local sin contrato publicado; la identidad
  del requisito monolítico se reemplaza por varios requisitos, sin referencias
  externas que actualizar.
