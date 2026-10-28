# Design

## Context

Ver proposal.md — Why. Estado actual relevante (verificado en el repo):

- El contrato vive en `swagger/swagger.yml` (Swagger 2.0) y se publica como
  estático bajo `/swagger` con `beego.SetStaticPath("/swagger", "swagger")`.
- El directorio `swagger/` ya contiene los assets de Swagger UI (`index.html`,
  `swagger-ui-*.js`, etc.), que por defecto cargan `swagger.json`.
- El proyecto ya depende de `gopkg.in/yaml.v3` (usado por el loader de
  relaciones) y de `encoding/json` de la biblioteca estándar.
- El arranque se orquesta en `main.go`, que aborta con `log.Fatal` ante errores
  de inicialización no recuperables (por ejemplo, configuración de relaciones
  inválida).

## Goals / Non-Goals

**Goals:**
- Mantener el YAML como única fuente de verdad y derivar el JSON de él.
- Conversión determinística e idempotente en disco, ejecutada al arranque.
- Fallo de arranque explícito si el contrato fuente falta o es inválido.

**Non-Goals:**
- No se genera el contrato a partir de anotaciones en los controladores
  (los handlers son `net/http` planos, no `beego.Controller` anotados).
- No se edita el contrato de forma programática ni se valida el esquema OpenAPI
  más allá de que el YAML sea parseable.
- No se añade un endpoint dinámico que renderice el JSON por petición: se
  materializa un archivo estático.

## Decisions

- **Reserialización 1:1 YAML→JSON con `yaml.v3` + `encoding/json`**: Swagger 2.0
  es el mismo modelo de datos en ambos formatos. Se decodifica el YAML a un
  valor genérico y se reserializa con `json.MarshalIndent`. `yaml.v3` decodifica
  los mapas con claves `string`, directamente compatibles con `encoding/json`,
  por lo que no se requiere conversión de tipos. Alternativa descartada:
  mantener un `swagger.json` escrito a mano — crea una segunda fuente de verdad
  que se desincroniza.
- **Generación al arranque, escrita en disco como estático**: el JSON se produce
  en `main.go` antes de registrar la ruta estática y se escribe junto al YAML en
  `swagger/`. Reutiliza la misma vía de publicación (`SetStaticPath`) para ambos
  formatos. Alternativa descartada: un handler que convierta por petición —
  añade trabajo por request y una ruta dinámica evitable para un contrato que
  cambia solo entre despliegues.
- **Idempotencia en disco (escribir solo si cambió)**: se compara el JSON
  derivado con el archivo existente y solo se reescribe ante diferencia. Evita
  tocar el archivo (y su mtime) en cada arranque sin necesidad.
- **Determinismo por indentación fija**: `json.MarshalIndent(doc, "", "  ")` con
  un salto final produce bytes estables para la misma entrada, de modo que dos
  arranques sin cambios en el YAML generan contenido idéntico.
- **Fallo explícito en arranque (`log.Fatal`)**: si el YAML no existe o no
  parsea, el API no inicia, coherente con el manejo de otros errores de
  inicialización no recuperables en `main.go`. Alternativa descartada: continuar
  sin publicar el JSON — dejaría a Swagger UI con un recurso ausente y
  enmascararía un contrato roto.

## Risks / Trade-offs

- [El directorio `swagger/` no se incluye en la imagen/entorno de ejecución] →
  la ruta estática no resuelve; mitigación: asegurar en el empaquetado que
  `swagger/` viaje junto al binario (verificable en el `Dockerfile`).
- [El JSON derivado queda desactualizado respecto al YAML] → se regenera en cada
  arranque desde el YAML, por lo que no puede divergir entre despliegues.
- [Diferencias de orden de claves entre formatos] → la equivalencia se define a
  nivel de modelo de datos, no de orden textual; los consumidores de OpenAPI no
  dependen del orden de claves de un objeto JSON.

## Migration Plan

1. Generar `swagger/swagger.json` al arranque desde `swagger/swagger.yml`.
2. Publicar ambos formatos vía la ruta estática `/swagger` ya existente.
3. Rollback: retirar la invocación del generador; el YAML y la ruta estática
   permanecen. Un `swagger.json` previo puede eliminarse si se desea.
