# F24: TerceroId no se verifica contra terceros

Estado: abierto. Reproducido en la corrida 4: P04 (`TerceroId` 999999) responde 200.

## Explicación
El proveedor se crea con `tercero_id = 999999`. Ese tercero no existe: `GET http://pruebasapi.intranetoas.udistrital.edu.co:8121/v1/tercero/999999` responde 404 (solo lectura). El contrato trata `TerceroId` como FK autoritativa del servicio de terceros.

## Propuesta
Al escribir la raíz, consultar `GET /v1/tercero/{TerceroId}` y devolver 400 si responde 404. Requiere decidir si el MID consulta terceros en cada escritura.
