# F09: el arranque del middleware depende de un catálogo remoto

Estado: abierto.

## Explicación
`main.go` resuelve `P_NATURAL` contra el CRUD de terceros antes de servir peticiones (`log.Fatalf` si falla). El host `pruebasapi.intranetoas.udistrital.edu.co:8121` es remoto. El 14/11 el primer arranque falló con timeout y el segundo funcionó, sin cambios de configuración.

## Propuesta
Comprobar el host antes de arrancar (GET de solo lectura a `/v1/tipo_contribuyente`). Si el arranque debe sobrevivir a una caída, cachear el catálogo o reintentar con límite de tiempo. Es un cambio de diseño (`API.md` pide fallo explícito al iniciar).
