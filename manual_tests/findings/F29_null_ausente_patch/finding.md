# F29: un campo nulo o ausente en un PATCH cambia el valor guardado sin que el cliente lo pida

Estado: cerrado_sin_solucion.
Solución: ninguna.
Gravedad: critica. Clasificación propuesta, ver utils/README.md.

## Explicación

- U11 (`PATCH /v1/proveedor` con `Id` 1 y `DescripcionPortafolio` null) responde 200 y guarda `descripcion_portafolio` como cadena vacía. La columna acepta NULL (`is_nullable = YES`). La matriz espera NULL.
- U05 (`PATCH` con `Id` 1 y un `Contacto` con `Id` 1 que no trae `Extension`) responde 200 y `contacto.extension` pasa de cadena vacía a NULL. El cuerpo no pidió ese cambio. La matriz solo espera el cambio de `valor_especifico`.

Observado en la corrida de la matriz del 14/11 (15:5x), con el MID de `/tmp/agora_mid_run_f04`.

## Propuesta

Definir el contrato para null y para campo ausente, en la raíz y en los hijos. Un campo ausente conserva su valor, como indica U10 para la raíz. Un null en un campo nullable se guarda como NULL. Pendiente de decisión.

## Causa compartida con F32
U11 y U05 tienen la misma causa de fondo: el modelo de `gestion_personas_crud` no puede representar NULL. Un NULL de la base llega al MID como valor cero o cadena vacía, y el PATCH lo reenvía. La causa y la propuesta de corrección están en F32, `## Causa` y `## Propuesta`.

## Decisión
Cerrado sin solución por el usuario (17/11/2026, 20:4x). La causa está en el modelo del CRUD, que no puede representar NULL (ver F32). El ajuste no debe hacerse en este repositorio.
