# F28: un PATCH con un Id que no existe crea un registro nuevo

Estado: abierto. Regresión introducida por F19 (14/11/2026). Reproducido en la corrida 3 y la corrida 4: U13 responde 200 y crea el proveedor 13 (`DescripcionPortafolio = "inexistente"`).

## Causa
Con F19, `Recuperar` devuelve `nil` cuando el CRUD responde 404 (`buscarRegistroExistente`). El motor clasifica el payload como creación. En la raíz, la creación descarta el `Id` del cliente y el CRUD asigna el siguiente de la secuencia. Antes de F19 el caso respondía 502.

Los hijos con PK compartida sí necesitan crear con el Id del padre (P06 y P19 dan 200 con `proveedor_natural.id = proveedor.id`), así que no se puede eliminar la creación con `Id`.

## Propuesta
Distinguir en el motor la raíz con `Id` del cliente de un hijo con PK compartida propagada por el padre. Para la raíz con `Id` ausente, responder 404 (o 400) sin escribir. Para el hijo, mantener la creación con el Id del padre. Pendiente de decisión (ver manual_tests.md, sección de propuestas).
