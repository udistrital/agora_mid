# Design

## Context

Ver `proposal.md` (Why). El estado relevante es puramente de especificación: dos
requisitos de specs principales superan 500 caracteres porque concentran varias
conductas normativas en una sola descripción. El comportamiento del sistema ya
está implementado y cubierto por pruebas; este cambio no toca código. La
restricción que moldea el enfoque es el contrato de OpenSpec para dividir un
requisito existente, que no es obvio y conviene fijar antes de sincronizar.

## Goals / Non-Goals

**Goals:**
- Dejar cada descripción de requisito por debajo de 500 caracteres (una conducta
  por requisito) sin perder ningún escenario ni alterar el significado normativo.
- Elevar la granularidad: pasar de 2 requisitos extensos a 7 atómicos que un
  lector pueda cargar y verificar de forma aislada.

**Non-Goals:**
- No se reubican físicamente los escenarios existentes fuera de su requisito
  original (ver Decisiones: el validador lo impide).
- No se cambia ningún comportamiento, firma ni contrato observable.
- No se tocan otros requisitos de ambas capacidades ni otras capacidades.

## Decisions

- **El `MODIFIED` conserva TODOS sus escenarios; la división se logra acortando
  su descripción y agregando requisitos `ADDED` con escenarios propios.**
  `openspec validate --strict` y `openspec archive` rechazan un `MODIFIED` que
  omita un escenario que el spec principal aún tiene (un `MODIFIED` reemplaza el
  bloque entero). Alternativa descartada: mover los escenarios al requisito nuevo
  y dejar el `MODIFIED` con menos — falla la validación. Por tanto, la cura del
  warning viene de recortar la *descripción* a una sola conducta; los requisitos
  nuevos documentan las conductas extraídas con escenarios redactados para ellos.
- **Partición de `nested-object-expansion`.** El `MODIFIED` "Selección de campos
  por rutas" retiene sus 12 escenarios y su descripción queda en una conducta
  (aceptar `fields` y resolver por último segmento). Se extraen tres requisitos:
  materialización de segmentos intermedios (esqueleto), recorte de lo no
  alcanzado, y precedencia por relación. Alternativa descartada: separar "aceptar
  `fields`" de "resolver por segmento" — son un solo acto y separarlos no añade
  claridad.
- **Partición de `api-response-contract`.** El `MODIFIED` "Extensión MetaData"
  retiene sus 11 escenarios y su descripción queda en `MetaData`/`Length`. Se
  extraen dos requisitos: la colección `Errors` y la obligación de `Kind`
  respaldado por constante (eje extensible). `Length` y `Errors` son ejes casi
  independientes y `Kind` tiene masa propia, de ahí el corte en tres.

## Risks / Trade-offs

- [El requisito original queda con una descripción corta pero aún con todos sus
  escenarios, por lo que la "atomicidad" es parcial: el bloque sigue siendo
  grande por número de escenarios] -> Mitigación: la carga cognitiva que se
  reducía era la de la descripción densa; los nuevos requisitos atómicos dan
  puntos de anclaje citables. Es el máximo que permite el contrato de OpenSpec
  sin perder escenarios.
- [Dividir cambia la identidad del requisito: donde había uno habrá varios] ->
  Aceptado: proyecto local sin contrato publicado ni referencias externas.
- [Posible solape conceptual entre el `MODIFIED` y un `ADDED`] -> Mitigación: la
  descripción del `MODIFIED` se acota a una sola conducta para no repetir la de
  los requisitos nuevos.

## Migration Plan

1. Sincronizar los delta specs a los specs principales (`/opsx-sync` o durante el
   archivado): merge semántico que aplica el `MODIFIED` acortado y agrega los
   requisitos `ADDED`.
2. Verificar `openspec validate --specs --strict` sin warnings de longitud.
3. Archivar el cambio.

Rollback: como no hay código, revertir es restaurar las descripciones originales
de ambos requisitos y eliminar los requisitos `ADDED` de los specs principales
(`git checkout` de los dos `spec.md` mientras el cambio no esté archivado).
