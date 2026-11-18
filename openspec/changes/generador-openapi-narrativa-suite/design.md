# Design

## Context

Ver proposal.md — Why. Hoy `swagger/swagger.yml` se autora a mano y `generarSwaggerJSON`
lo reserializa 1:1 a `swagger/swagger.json` en cada arranque, escribiendo solo si el
contenido cambió; `main` aborta el arranque si esa derivación falla. El cambio de
composición de objetos tipados introduce la tabla de campos tipada con un vocabulario
cerrado de tipos de almacenamiento y deja explícitamente abierto cómo esos tipos se
renderizan en documentación. Este cambio cierra ese mapeo y antepone una etapa de
generación `config -> swagger.yml` a la derivación existente.

## Goals / Non-Goals

**Goals:**
- Invertir la fuente de verdad: la configuración genera `swagger.yml`.
- Preservar intacta la derivación `swagger.yml -> swagger.json` y la publicación en
  ambos formatos.
- Fijar el mapeo tipo de almacenamiento -> tipo/formato OpenAPI.
- Mantener la narrativa `test/tests.md` coherente con el árbol #58 y el contrato de
  respuesta.

**Non-Goals:**
- No se incluyen pruebas ejecutables del comportamiento del API; viven en los
  cambios que prueban ese comportamiento. Este cambio aporta el generador y la
  narrativa.
- No se cambia la reserialización 1:1 a JSON ni el aborto de arranque de esa etapa.
- No se define la proyección inversa ni ningún servicio nuevo (por ejemplo
  cotizaciones); solo el mapeo y la generación del contrato vigente.

## Decisions

### Cadena determinística de tres etapas, regenerada en cada arranque

```
   config (objetos direccionables + tabla de campos tipada + arbol efectivo)
        |   <-- fuente de verdad
        v   generarSwaggerYAML(config)      [NUEVO]   regenera cada arranque, escribe-si-cambio
        |
   swagger/swagger.yml   (caché generada, versionada)
        |
        v   generarSwaggerJSON()            [EXISTE]  regenera cada arranque, escribe-si-cambio
        |
   swagger/swagger.json  (caché generada, versionada)
        |
        v   Swagger UI en /swagger
```

La nueva etapa se antepone a la existente en `main`, antes de `generarSwaggerJSON`.
Se eligió generar `swagger.yml` (y no OpenAPI directamente en ambos formatos) porque
reutiliza sin cambios toda la lógica de derivación a JSON, el aborto de arranque y la
publicación en ambos formatos ya probados: la etapa nueva solo produce la entrada de
esa cadena. Alternativa descartada: generar JSON y YAML por separado desde la
configuración duplicaría la serialización y dos caminos de publicación.

### Determinismo byte-estable de la etapa config -> YAML

El idempotente-en-disco solo se sostiene si la generación es byte-estable: orden
estable de claves de objeto, de campos y de colecciones. Sin esto, cada arranque
reescribiría `swagger.yml` (y en cascada `swagger.json`), produciendo diffs de git
ruidosos — el mismo problema que la etapa JSON ya evita. Por eso el determinismo de
la etapa `config -> YAML` se afirma explícitamente, no se hereda.

### Mapeo de tipo de almacenamiento a tipo/formato OpenAPI

El cambio de objetos tipados fijó el vocabulario de almacenamiento y dejó el mapeo a
documentación para este cambio. El mapeo propuesto (Swagger 2.0, coherente con el
contrato existente):

| Almacenamiento | type (OpenAPI) | format      |
|----------------|----------------|-------------|
| boolean        | boolean        | —           |
| integer        | integer        | int64       |
| numeric        | number         | —           |
| string         | string         | —           |
| uuid           | string         | uuid        |
| timestamp      | string         | date-time   |
| jsonb          | object         | —           |

`integer` se mapea a entero para que las llaves foráneas y PK compartidas se
describan como números, coherente con el contrato de escritura que las materializa
numéricas. `jsonb` se describe como objeto por ser estructura libre.

### Fuente de fallo desplazada a la configuración

El aborto de arranque ante configuración inválida ya lo provee la composición de la
configuración (cambio de objetos tipados). Una configuración válida garantiza un
`swagger.yml` generable; si aun así el YAML generado no derivara a JSON, el aborto de
esa etapa existente se conserva. No se añade un aborto redundante por "YAML fuente
ausente" porque el YAML deja de ser fuente autorada.

## Risks / Trade-offs

- [`swagger.yml` versionado y generado puede quedar desincronizado si alguien lo
  edita a mano] → La regeneración en cada arranque lo sobreescribe desde la
  configuración; la narrativa y la documentación indican que es un artefacto
  generado, no editable.
- [El mapeo de tipos congela decisiones de documentación en este cambio] → Se
  concentra en un único lugar (el generador) y se cubre con pruebas por tipo, de modo
  que un ajuste futuro es local y verificable.
- [B ya modifica el requisito de endpoints de escritura de `openapi-publication`] →
  Este cambio no toca ese requisito; modifica solo los de fuente de verdad y añade la
  generación y el mapeo, de modo que ambos cambios componen sin solaparse en el mismo
  requisito.

## Migration Plan

En la fase apply, `swagger.yml` pasa de fuente autorada a artefacto generado: se
añade el generador y se antepone en `main` a la derivación JSON existente; el primer
arranque regenera `swagger.yml` desde la configuración y, si difiere, actualiza
`swagger.json`. `test/tests.md` se reescribe a la suite #58. No hay migración de
datos ni cambio en la derivación JSON.
