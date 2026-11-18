# F14: el middleware reporta "profundidad reducida" para casi todas las relaciones

Estado: aceptado. Decisión de comportamiento tomada por el usuario (17/11/2026, 19:27): una reducción de profundidad que no corta hijos no emite aviso. El texto de especificación está pendiente.
Solución: aplicada_sin_commit. Corrección rápida en el árbol de trabajo. Ver `## Cambio aplicado`.
Gravedad: media. Clasificación propuesta, ver utils/README.md.

## Explicación
`POST /v1/proveedor/consulta {"id":1}` respondía 206 con 17 avisos `kind: depth` ("profundidad reducida a N por configuración de la relación"). Esos avisos aparecían aunque ningún hijo quedara sin expandir. El 206 hacía que la respuesta pareciera degradada.

## Propuesta
No emitir `depth` por la sola reducción de profundidad. Un aviso `depth` solo aparece cuando el límite corta una relación hija pedida. La ruta del aviso y su texto están en F31.

## Cambio aplicado

**Archivos modificados:**

- `helper/expand.go`: se quitó el aviso por reducción de profundidad en `expandirRelacionNivel`.
- `helper/expand_test.go`: `TestExpandSinAvisoDepthSiLaReduccionNoCortaHijos`, `TestExpandSinAvisoDepthSiLaRelacionNoTieneHijos` y `TestExpandSinAvisoDepthSinRegistros`.

**Verificación:** `go build ./...` y `go test -count=1 ./...` pasan. En vivo, `POST /v1/proveedor/consulta {"id":1}` (19:3x) bajó de 17 avisos a 4. Los 4 son cortes reales de hijas (ver F31). Ninguno es por reducción sin corte.

## Notas de especificación

Estado de la especificación: sin cambios.

Conflicto: el escenario actual "Configuración hija con menor profundidad" de `openspec/specs/nested-object-expansion/spec.md` dice que se agrega un aviso `depth` al reducir la profundidad. El código ya no lo hace.

Texto propuesto para reemplazar ese escenario:

```
#### Scenario: Configuración hija con menor profundidad sin corte
- **WHEN** una configuración hija declara una profundidad menor que el límite restante y no impide expandir ninguna relación hija pedida
- **THEN** la rama continúa con el valor menor y no se agrega aviso de tipo `depth`
```

Pendiente: aplicar el texto por separado. El escenario de corte con ruta está en F31.

## Decisión
Aprobado por el usuario (17/11/2026, 20:20). Una reducción de profundidad que no corta hijos no emite aviso `depth`. El escenario "Configuración hija con menor profundidad sin corte" de `nested-object-expansion` se aplica por separado, con aprobación del texto final.
