# F17: guarda de solo lectura para terceros, parámetros y ubicaciones

Estado: aceptado. Pruebas unitarias en verde.
Solución: aplicada_sin_commit. Implementado en el árbol de trabajo (sin commit). Ver `## Cambio aplicado`.
Gravedad: alta. Clasificación propuesta, ver utils/README.md.

## Explicación
Sin guarda, una relación a `terceros`, `parametros` o `ubicaciones` podía crear o modificar datos ajenos, porque todas las peticiones salen por `clients.HTTPClient.do`.

## Solución aplicada
- `clients/solo_lectura.go`: registro de destinos protegidos (esquema, host y ruta). `ProtegerSoloLectura` rechaza una base vacía o inválida.
- `clients/http_client.go`: `do` rechaza cualquier método distinto de GET hacia un destino protegido, antes de abrir la conexión. El cliente por defecto revisa cada redirección.
- `main.go`: registra `terceros`, `parametros` y `ubicaciones` al arrancar y aborta si falta una URL.
- Pruebas: `clients/solo_lectura_test.go`.

## Conflicto abierto
`openspec/specs/proveedor-write` y `declarative-write-engine` exigen crear y actualizar `tercero` y `datos_identificacion`. Con la guarda, esas escrituras fallan. Hay que elegir: (1) mantener la guarda total y dejar la escritura de `tercero` pendiente, o (2) permitir escritura solo a `tercero` y `datos_identificacion` con una lista explícita, lo que requiere cambiar la especificación.

## Riesgos residuales
- La protección depende de que `main` la registre. Las pruebas con `httptest` no la activan.
- Un `HTTPClient` con un `Client` propio no pasa por la revisión de redirecciones.

## Cambio aplicado

**Archivos modificados:**

- `clients/http_client.go`.
- `main.go`.
- `clients/solo_lectura.go` (nuevo).
- `clients/solo_lectura_test.go` (nuevo).

**Verificación:** `go test ./clients/...` pasa. Las pruebas de `clients/solo_lectura_test.go` cubren la guarda.

## Decisión

Aceptado. La guarda de solo lectura sobre terceros, parámetros y ubicaciones se mantiene durante las pruebas. Es un quirk de la metodología de prueba, no un requisito del producto.

Efecto conocido: mientras la guarda esté activa, un alta con objeto `Tercero` queda bloqueada por `ErrEscrituraBloqueada`. Esa es la regresión R4 de la revisión del 14/11. Se retirará la guarda (quitar `clients.ProtegerSoloLectura` de terceros en `main.go`) cuando se pruebe la escritura a terceros en un entorno local.

No se reprodujo en vivo, a propósito: la prueba sería una escritura hacia el CRUD de terceros si la guarda fallara. La protección queda cubierta por `clients/solo_lectura_test.go`.
