# Tasks

## 1. Refinar nested-object-expansion

- [x] 1.1 En el delta `specs/nested-object-expansion/spec.md`, acortar la
  descripción del requisito MODIFIED "Selección de campos por rutas" a una sola
  conducta (aceptar `fields` y resolver por último segmento) conservando sus 12
  escenarios; verificar que la descripción queda < 500 caracteres y que
  `openspec validate refinar-granularidad-specs --strict` no reporta ese warning
- [x] 1.2 En el mismo delta, agregar los requisitos ADDED "Materialización de
  segmentos intermedios como esqueleto", "Recorte de campos y relaciones no
  alcanzados" y "Precedencia por relación en la selección de campos", cada uno con
  su propio escenario; verificar con `openspec validate refinar-granularidad-specs
  --strict` que cada requisito tiene al menos un escenario y pasa en estricto

## 2. Refinar api-response-contract

- [x] 2.1 En el delta `specs/api-response-contract/spec.md`, acortar la
  descripción del requisito MODIFIED "Extensión MetaData para avisos y total de
  listas" a `MetaData`/`Length` conservando sus 11 escenarios; verificar que la
  descripción queda < 500 caracteres en `openspec validate
  refinar-granularidad-specs --strict`
- [x] 2.2 En el mismo delta, agregar los requisitos ADDED "Colección Errors de
  avisos en MetaData" y "Tipo de aviso Kind respaldado por constante declarada",
  cada uno con su propio escenario; verificar con `openspec validate
  refinar-granularidad-specs --strict` que la validación completa pasa sin errores
  ni warnings de longitud

## Workflow follow-up

- Sincronizar los delta specs a los specs principales con `/opsx-sync
  refinar-granularidad-specs` (merge semántico: aplica el MODIFIED acortado y
  agrega los requisitos ADDED en ambas capacidades).
- Tras sincronizar, verificar `openspec validate --specs --strict` sin warnings de
  longitud en `nested-object-expansion` ni `api-response-contract`.
- Archivar el cambio con `/opsx-archive refinar-granularidad-specs`.
