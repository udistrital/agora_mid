# F29: un campo nulo o ausente en un PATCH cambia el valor guardado sin que el cliente lo pida

Estado: abierto.
Solución: ninguna.
Gravedad: critica. Clasificación propuesta, ver utils/README.md.

## Explicación

- U11 (`PATCH /v1/proveedor` con `Id` 1 y `DescripcionPortafolio` null) responde 200 y guarda `descripcion_portafolio` como cadena vacía. La columna acepta NULL (`is_nullable = YES`). La matriz espera NULL.
- U05 (`PATCH` con `Id` 1 y un `Contacto` con `Id` 1 que no trae `Extension`) responde 200 y `contacto.extension` pasa de cadena vacía a NULL. El cuerpo no pidió ese cambio. La matriz solo espera el cambio de `valor_especifico`.

Observado en la corrida de la matriz del 14/11 (15:5x), con el MID de `/tmp/agora_mid_run_f04`.

## Propuesta

Definir el contrato para null y para campo ausente, en la raíz y en los hijos. Un campo ausente conserva su valor, como indica U10 para la raíz. Un null en un campo nullable se guarda como NULL. Pendiente de decisión.
