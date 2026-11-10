# Design

## Context

Ver proposal.md — Why. El motor de escritura genérico ya recorre el árbol efectivo
en orden topológico y, por el control de escritura por relación, ejecuta un pre-pass
de solo lectura que valida el árbol completo antes de materializar: una validación
fallida responde `400` sin ninguna escritura. El cambio que absorbe el modelo #58
define los casos de reconciliación (incluida la transición de `estado_proveedor`:
crear el nuevo estado antes de dar de baja el previo). Lo que falta es el contrato
de lo que ocurre cuando la cascada ya pasó el pre-pass y una dependencia remota
falla a mitad de materialización.

## Goals / Non-Goals

**Goals:**
- Fijar un mapa de códigos de estado inequívoco para la escritura, anclado al
  momento del fallo en la línea de tiempo.
- Definir el `206` de stop-abort: subárbol parcial en `Data`, punto de parada y
  causa en `MetaData.Errors`.
- Fijar la identidad de reconciliación que hace seguro el reintento, incluida la
  transición de estado.
- Fijar la distinción omisión / vacío / null del contrato de escritura.

**Non-Goals:**
- Ninguna transacción distribuida ni rollback entre CRUD dueños. La recuperación es
  por reintento idempotente, no por atomicidad.
- No se redefine la validación previa (`400`) ni el pre-pass; son frontera heredada.
- No se infiere ninguna eliminación por diferencia contra el estado del CRUD.

## Decisions

### La línea de tiempo de la escritura define el código de estado

```
   request
     |
     v
   [ pre-pass: resolver + validar el arbol completo ]   (solo lectura)
     |      \__ validacion falla         -> 400   (frontera: control por relacion)
     |      \__ dependencia inalcanzable -> 502 / 504  (nada materializado)
     | validacion ok, dependencias alcanzables
     v
   [ materializar nodo 1 ] ok
   [ materializar nodo 2 ] ok
   [ materializar nodo 3 ] --X fallo/timeout de dependencia
     |                         -> 206 stop-abort
     v                            Data = {1,2}; MetaData.Errors nombra nodo 3
   [ nodos 4,5 no se intentan ]
     |
     v
   cascada completa sin avisos -> 200
   cascada completa + canonizacion -> 206 (field_normalization)
```

**Por qué `206` y no `502`/`504` para el fallo a mitad de cascada.** Un `206`
comunica con honestidad "empecé a escribir y me detuve: esto quedó materializado y
aquí se rompió". Esa semántica solo tiene sentido cuando existe un subárbol parcial
que entregar. Un fallo de dependencia durante el pre-pass de solo lectura no produjo
ninguna escritura: no hay nada parcial que devolver, así que un `206` sería
engañoso. Por eso `502`/`504` queda reservado, de forma estrecha, al fallo de
dependencia *previo a toda escritura*, y `206` cubre el stop-abort *después* de
materializar al menos una entidad. Alternativa descartada: usar `502`/`504` también
a mitad de cascada perdería el subárbol parcial y la identidad de reconciliación que
habilita el reintento.

### La taxonomía de fallo se expresa por `Kind`, sin segundo eje

El envoltorio ya distingue `field_normalization` de un fallo remoto tolerado
mediante `Kind` sobre un conjunto abierto respaldado por constantes. La distinción
validación / dependencia / timeout extiende ese mismo conjunto
(`dependency_failure`, `timeout`) en lugar de añadir un campo discriminador
paralelo. Alternativa descartada: un eje "category" separado de `Kind` duplicaría la
clasificación y obligaría a mantener dos vocabularios coherentes.

### Reconciliación por re-find, no por huella de valor

El reintento vuelve a enlazar cada nodo por `Id` y, en su defecto, por clave
natural. Las colecciones sin `Id` ni clave natural solo son idempotentes por la vía
de reenvío del `Data` materializado (donde el elemento ya trae su `Id`); el contrato
declara explícitamente que reenviar el cuerpo original puede duplicarlas. Esto honra
el criterio de aceptación de no asumir idempotencia de colecciones nuevas sin clave
estable. Alternativa descartada: deduplicar por huella de los campos escribibles —
frágil (cualquier cambio de campo parece un elemento nuevo) y no soportado por los
cambios previos. La transición de estado es un caso de re-find por `estado_id`, no un
mecanismo aparte.

### Omisión / vacío / null por tipo de campo

| Entrada            | Escalar escribible                 | FK relacional derivada          | Colección                         |
|--------------------|------------------------------------|---------------------------------|-----------------------------------|
| Ausente            | conserva el valor existente        | conserva / deriva               | conserva elementos; no elimina    |
| `null`             | fija null si nullable; `400` si no | tratar como ausente, derivar    | tratar como ausente               |
| Vacío (`""`, `[]`) | valor literal                      | n/a                             | sin elementos afirmados; no elimina|

El `null` es keyed por tipo de campo porque una FK derivada tiene una llave
autoritativa que el motor puede calcular (tratar `null` como ausente es correcto),
mientras que un escalar nullable sí admite `null` como valor real. `[]` nunca
elimina: el motor solo materializa bajas de nodos marcados por el árbol.

## Risks / Trade-offs

- [Un cliente que reenvía el cuerpo original tras un `206`, no el `Data`, duplica
  colecciones sin clave] → El contrato documenta la vía segura (reenviar el `Data`)
  y la narrativa de pruebas de D cubre ambos caminos; la limitación es explícita, no
  silenciosa.
- [Distinguir `dependency_failure` de `timeout` depende de que el cliente HTTP
  propague la causa (timeout vs. error de conexión)] → Se mapea a partir del tipo de
  error del cliente; si la causa no es distinguible, el aviso cae en
  `dependency_failure` como caso general.
- [`502`/`504` solo para el pre-pass estrecha su alcance frente a lo que algunas
  herramientas esperan] → Es deliberado: preserva el significado de `206` como
  "materialización parcial" y mantiene vigente el listado `502/504` del contrato
  publicado para el fallo previo a la escritura.

## Migration Plan

No aplica como migración de datos. En la fase apply, el envoltorio y el motor emiten
el `206` de stop-abort con el aviso tipado y se declaran las constantes de `Kind`
`dependency_failure` y `timeout`; el comportamiento previo (`400` en validación,
`200` en éxito) no cambia.
