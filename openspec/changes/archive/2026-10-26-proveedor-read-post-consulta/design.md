# Design

## Context

Ver `proposal.md` (Why) para la motivación. Estado actual relevante:

- `/v1/proveedor` se registra con `beego.Any` y se enruta por método en `main.go` (`enrutarProveedor`): `POST`/`PATCH` -> `Escribir` (escritura), cualquier otro método -> `Leer` (lectura actual por query string).
- El manejador `Leer` (`controllers/proveedor.go`) lee `id` y `fields` del query string, parte `fields` por comas y valida cada ruta antes de invocar `servicio.LeerProveedor(ctx, id, fields)`.
- La validación canónica estricta ya existe en `helper.ParsearFields`: compara cada segmento con `==` contra `allowed_fields`/`nested_relations` y rechaza rutas vacías, duplicadas o no permitidas con `*helper.ErrorFields` -> 400. No normaliza mayúsculas/espacios.
- El servicio arma la URL del CRUD dueño en el código (`services/proveedor.go`: `base + "/v1/proveedor/"`), no a partir del path entrante.
- WSO2 enruta por configuración JSON ligada a `swagger/swagger.yml`. No hay nada publicado; todo es local. WSO2 no admite `:` ni `_` en el path.

## Goals / Non-Goals

**Goals:**
- Exponer la lectura como `POST /v1/proveedor/consulta` con `id` y `fields` (arreglo JSON) en el cuerpo, conservando el contrato de respuesta y la firma del servicio.
- Permitir coexistencia temporal de GET y POST durante la implementación para que el microfrontend migre componente a componente.
- Retirar el GET de lectura como último grupo de trabajo, dejando `405` para métodos no soportados.
- Dejar la convención `{recurso}/consulta` lista para otros recursos (p. ej. terceros).

**Non-Goals:**
- No se normalizan nombres de campos no canónicos; se rechazan (ver spec `proveedor-read`).
- No cambia la firma `LeerProveedor(ctx, id, []string)` ni la lógica de expansión.
- No cambia la escritura `POST`/`PATCH` sobre `/v1/proveedor`.
- No se modifica a nivel de requisitos la capacidad `openapi-publication`; solo se edita el contenido de `swagger/swagger.yml`.

## Decisions

- **Path `POST /v1/proveedor/consulta` (Forma 2a), segmento de acción plano.** Alternativas: `POST /v1/proveedor` con discriminador en el cuerpo (colisiona con la escritura sobre el mismo verbo+path), `/v1/consulta/proveedor` o un `recurso` en el cuerpo (namespace de acción separado), y las formas `proveedor:consulta` / `proveedor/_consulta`. Se descartan las dos últimas porque WSO2 no admite `:` ni `_`. El riesgo de que `consulta` se interprete como un sub-recurso del CRUD no se materializa en el MID: el servicio arma la URL del CRUD en código y no reenvía el segmento entrante. Se elige 2a por legibilidad, por conservar el recurso en el path (políticas de WSO2 por recurso siguen aplicando) y por consistencia con la convención `{recurso}/consulta`.
- **`fields` como arreglo JSON.** Alternativa: cadena separada por comas en el cuerpo. Se elige el arreglo porque es el objetivo de mover la selección al cuerpo y elimina el parseo por comas y las preocupaciones de codificación/normalización; la validación posterior reutiliza `helper.ParsearFields`.
- **Nueva ruta y manejador dedicados.** Se registra una segunda ruta `/v1/proveedor/consulta` (beego no hace coincidir `/v1/proveedor` con el subpath) y un manejador `Consultar` que decodifica el cuerpo, valida `id` y `fields` reproduciendo las validaciones de `Leer`, y delega en `LeerProveedor`. El enrutado raíz `/v1/proveedor` se conserva para la escritura.
- **Coexistencia temporal y retiro final.** El POST se implementa primero y GET permanece durante la migración del microfrontend. El último grupo de tareas elimina `Leer`, la rama por defecto de `enrutarProveedor` que atendía lectura, el path `GET /v1/proveedor` en `swagger.yml` y las pruebas específicas de GET. Tras el retiro, `/v1/proveedor` responde `405` a `GET`/`PUT`/`DELETE` (solo `POST`/`PATCH` escritura), y `/v1/proveedor/consulta` responde `405` a todo lo que no sea `POST`.

## Risks / Trade-offs

- [POST tiene menos ventajas de caché que GET] -> Aceptado: la estabilidad de la URL y evitar los límites de WSO2 pesan más para una consulta con selección de campos extensa.
- [El cuerpo también puede quedar registrado en logs según la configuración] -> Igual que con GET; no se exponen credenciales en el cuerpo (solo `id` y nombres de campos).
- [Ventana de coexistencia GET+POST deja dos caminos de lectura temporalmente] -> Mitigación: el retiro de GET es el grupo final obligatorio del cambio; la validación de cierre verifica que no quede GET de lectura ni en código ni en `swagger.yml`.
- [WSO2 podría transformar el cuerpo JSON al reenviar] -> El supervisor pide confirmarlo en ambiente real; como todo es local aún, el diseño asume que WSO2 reenvía el cuerpo sin transformar y la verificación en gateway queda como tarea de despliegue, no bloqueante para la especificación.

## Migration Plan

1. Implementar `POST /v1/proveedor/consulta` (ruta + manejador `Consultar` + decodificación/validación de cuerpo) manteniendo GET.
2. Publicar `POST /v1/proveedor/consulta` en `swagger/swagger.yml`.
3. El microfrontend migra sus llamadas de lectura al POST, componente a componente.
4. Grupo final: retirar el GET de lectura (código, enrutado, `swagger.yml`, pruebas) y dejar `405` para métodos no soportados.

Rollback: mientras el GET exista (pasos 1-3), revertir es volver a apuntar el microfrontend al GET. Tras el paso 4 el rollback implica reintroducir el GET; por eso el retiro es el último grupo, una vez confirmada la migración.
