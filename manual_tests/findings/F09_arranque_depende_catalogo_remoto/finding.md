# F09: el arranque del middleware depende de un catálogo remoto

Estado: resuelto.
Solución: ninguna. Ver `## Propuesta`.
Gravedad: alta. Clasificación propuesta, ver utils/README.md.

## Explicación
`main.go` resuelve `P_NATURAL` contra el CRUD de terceros antes de servir peticiones (`log.Fatalf` si falla). El host del catálogo de terceros es remoto. El 14/11 el primer arranque falló con timeout y el segundo funcionó, sin cambios de configuración.

## Propuesta
Comprobar el host antes de arrancar (GET de solo lectura a `/v1/tipo_contribuyente`). Si el arranque debe sobrevivir a una caída, cachear el catálogo o reintentar con límite de tiempo. Es un cambio de diseño (`API.md` pide fallo explícito al iniciar).

## Cambio aplicado

Sin cambio de código. El arranque no se considera un error: el MID depende del catálogo de terceros por diseño, y el fallo es esperable cuando el host no responde.

**Archivos modificados:**

- Ninguno.

**Verificación:** el MID arranca con el catálogo de terceros disponible (log del 14/11, 15:20, PID 87903: "http server Running" sin errores de `P_NATURAL`).

## Notas de especificación

El origen de `P_NATURAL` va a cambiar. Hoy se resuelve con el tipo de contribuyente del CRUD de terceros. Pasará a resolverse con `tipo_proveedor`, que está en otra tabla. El valor también puede cambiar. Resuelto, no aprobado: la nota queda para programarla con la sincronización de la especificación.
