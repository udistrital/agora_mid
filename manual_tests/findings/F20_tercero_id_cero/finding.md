# F20: un proveedor sin TerceroId se guarda con tercero_id = 0

Estado: abierto. Confirmado en la corrida 3 (14/11): los proveedores creados sin `TerceroId` quedan con `tercero_id = 0` (P01, P02, P05 y los demás casos sin `TerceroId`).
Solución: propuesta. Ver `## Propuesta`.
Gravedad: alta. Clasificación propuesta, ver utils/README.md.

## Explicación

Un proveedor creado sin `TerceroId` se guarda con `tercero_id = 0`. Ver `## Causa`.

## Causa
`models/proveedor.go` del CRUD declara `TerceroId int`. Go envía 0 cuando el campo falta, y el ORM lo escribe. La columna es nullable, y 0 no existe en terceros: queda como FK huérfana.

## Propuesta

En revisión. Hay dos opciones; ninguna está aplicada.

- **Opción A (recomendada, cambio en el CRUD):** declarar `TerceroId *int` en `models/proveedor.go`, para que un campo ausente quede NULL y no 0. Cumple `API.md:197`, que dice que `TerceroId` puede omitirse. El MID debe excluir 0 de la clave natural y tratar 0 como ausente en la lectura (ver Análisis).
- **Opción B (cambio en el MID y en la especificación):** rechazar con 400 un alta raíz (POST) sin `TerceroId`. Evita la fila con 0 desde el MID. Contradice `API.md:197` ("puede omitirse"), así que requiere aprobar antes un cambio en `openspec/specs/proveedor-write`.

Corrección a la propuesta anterior: "no enviar `TerceroId` cuando sea 0" no funciona. El CRUD decodifica un campo ausente como 0 y lo guarda igual, así que el MID no puede evitarlo solo omitiendo el campo.

## Análisis: CRUD o MID (14/11, 14:17)

**Causa de fondo: el CRUD.** El modelo `models/proveedor.go` declara `TerceroId int` sin `null`. La columna `proveedores.proveedor.tercero_id` es nullable. Cuando el cuerpo no trae el campo, Go deja 0 y el ORM lo escribe. El MID no envía 0 cuando el campo falta: el cuerpo de P01 no lo incluye.

**Dónde participa el MID.**
- Lectura: `services/proveedor.go` declara `TerceroId int`. Un NULL llega como 0, así que la respuesta no distingue "sin tercero" de 0.
- Clave natural: `helper/write_engine.go` (`valoresClaveNatural`, línea 1371) toma cualquier valor no nulo, incluido 0. Un POST con `TerceroId` 0 busca `TerceroId:0,Activo:true` y puede reutilizar un proveedor creado sin tercero y guardado como 0. Hipótesis de código, no reproducida en vivo.
- Índice: `idx_proveedor_tercero_id` no es único en la base, aunque los comentarios del MID hablan de clave única.

**Si el CRUD permitiera NULL (`TerceroId *int`).**
- Escritura sin `TerceroId`: no cambia (la columna queda NULL en lugar de 0).
- Escritura con `TerceroId` null explícito (P05): quedaría NULL. Hoy queda 0.
- Lectura: el MID debe cambiar su modelo a puntero, o documentar que 0 significa "sin tercero".
- Clave natural: con NULL no hay colisión con 0. El MID debe excluir 0 de `valoresClaveNatural`, según la regla de ids en cero. Eso es un cambio del MID, aunque el CRUD cambie.
- Índice: para que la clave natural se haga cumplir, haría falta un índice único parcial (`WHERE tercero_id IS NOT NULL`). En PostgreSQL, NULL no choca con NULL en un índice único.
- Datos existentes: habría que convertir las filas con 0 a NULL. Hoy no hay filas con 0 ni NULL (restauración del 14/11), así que no se puede comprobar el alcance en la base actual.

**Límite.** El CRUD está en `../gestion_personas_crud`, fuera de este repositorio. No se modifica desde estas pruebas. El cambio de modelo, la migración y el índice los decide el equipo del CRUD.

**Verificación en vivo (14/11, 14:55):** P05 (`TerceroId` null) responde 200 y crea el proveedor 3 con `tercero_id` 0 (`proveedor` 3 `|0`). Se restauró el respaldo después.
