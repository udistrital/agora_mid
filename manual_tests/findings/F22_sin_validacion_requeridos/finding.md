# F22: no se validan los campos requeridos de un hijo

Estado: aceptado. Reproducido en la corrida 4: P11 responde 200.
Solución: aplicada_sin_commit. Ver `## Propuesta`.
Gravedad: alta. Clasificación propuesta, ver utils/README.md.

## Explicación
`POST /v1/proveedor` con un `Contacto` sin `Proposito` crea el contacto con `proposito = ''`. El contrato pide 400 o 502 para un campo requerido ausente.

## Causa
Ni el MID ni el CRUD validan campos requeridos de los hijos. La base acepta la cadena vacía.

## Propuesta
Validar en el MID los campos `required` del objeto antes de escribir, y devolver 400 con el nombre del campo. Nota: la configuración de `required` debería venir de un sidecar en el generador de configuración, no de una lista escrita a mano en el MID.

## Análisis: configuración y requeridos (14/11, 14:23)

**Lo que contiene hoy la configuración.** Cada campo tiene `nullable` y `required`:
- `nullable` sí está poblado. Viene del DBML: `false` si la columna es `NOT NULL` o PK. Hay 72 campos no nulables y 54 nulables.
- `required` existe, pero no está poblado: los 126 campos tienen `required: false`. El generador (`utils/generador_configuracion/README.md`, línea 72) dice que no deriva la obligación del cliente y la deja sin poblar.

**Derivación posible sin cambiar el generador.** Un campo escribible, no nulable y sin default en la base es requerido para el cliente. Los defaults de la base solo existen en `id`, `activo` y `fecha_creacion`, que no son escribibles. Según esa regla, los requeridos escribibles son:
- Contacto: `ValorEspecifico`, `TipoCanalContactoId`, `Proposito`.
- CuentaBancaria: `TipoCuentaId`, `NumeroCuenta`, `EntidadBancariaId`.
- Documento: `TipoDocumentoId`, `Enlace`.
- ActividadEconomicaProveedor: `ActividadEconomicaId`.
- PerfilProveedor: `PerfilId`.
- PersonaJuridicaTipoEntidad: `TipoEntidadId`.
- Proveedor: `TipoProveedor`.
- ProveedorNatural: `PaisNacimientoId`.
- ProveedorJuridico: `ProcedenciaEmpresaId`, `TipoConformacionId`, `ExclusividadProducto`, `RegimenContributivo`, `Pyme`, `RegistroMercantil`, `SujetoRetencion`, `AgenteRetenedor`, `ResponsableIca`, `ResponsableIva`.
- Representacion: `RepresentanteId`.
- TituloProveedorNatural: `TituloObtenido`, `NucleoBasicoConocimientoSniesId`, `PerfilId`.

Las FK que llena el padre (`ProveedorId`, por ejemplo) no son escribibles y no aparecen en la lista.

**Por qué P11 pasa hoy.** La restricción `NOT NULL` no rechaza un valor vacío. Si el cuerpo omite `Proposito`, Go deja `""` y la base lo acepta. Lo mismo ocurre con los enteros (0) y los booleanos (false). Por eso la validación del MID tiene que mirar la presencia en el cuerpo (clave ausente o `null`), no solo la restricción de la base. Si se quiere rechazar también la cadena vacía, es una decisión aparte.

**Matices para programar.**
- En creación, los requeridos se exigen completos. En PATCH, un campo omitido conserva su valor (U10), así que la regla debe aplicarse solo a campos enviados como `null` o a creaciones.
- Los booleanos requeridos, como `Pyme`, no tienen forma de expresar ausencia en Go. Hay que decidir si omitirlos es error.
- La fuente de la regla puede ser la configuración (derivando desde `nullable`, `writable` y la ausencia de default) o poblar el campo `required` en el generador. Esta última opción toca el generador, que vive en este repositorio.

## Cambio aplicado

Solución sucia sin commit (17/11/2026, 20:3x), a pedido del usuario.

**Archivos modificados:**

- `helper/requeridos_hijos.go` (nuevo): `requeridosCreacionHijo` (lista escrita a mano: `Contacto` exige `ValorEspecifico`, `Proposito` y `TipoCanalContactoId`; `CuentaBancaria` exige `TipoCuentaId`, `NumeroCuenta` y `EntidadBancariaId`) y `validarRequeridosCreacion`, que responde `*ErrorEscritura` con la ruta del campo faltante.
- `helper/write_engine.go`: `validarRelacionAnidada` llama a `validarRequeridosCreacion` para cada hijo, en el caso de objeto y en el de lista.
- `helper/requeridos_hijos_test.go` (nuevo): pruebas de creación sin campo, actualización por Id (no exige campos), `Id` 0 como creación, `CuentaBancaria` sin entidad, hijo completo, y relación sin requeridos.
- `services/proveedor_write_test.go`: `TestEscribirProveedorCreacionCompleta` enviaba un contacto sin `Proposito` ni `TipoCanalContactoId`. Se agregaron esos campos al cuerpo, y `WritableFields` del fixture de contacto pasó a incluirlos, porque el fixture modela el contacto real.

**Verificación:** `go build ./...` y `go test -count=1 ./...` pasan en todos los paquetes. La prueba `helper/requeridos_hijos_test.go` se escribió primero, falló al compilar (función inexistente) y luego pasó.

**Verificación en vivo (17/11/2026, 20:3x)**, MID reiniciado por el tester con el binario corregido (PID 171539), con respaldo `snapshots/proveedores_antes_f22_17nov.sql` y restauración al final:
- P11 (contacto sin `Proposito`): HTTP 400, mensaje `cuerpo de escritura inválido en "Contacto[0].Proposito": campo requerido ausente en la creación`. Conteos sin cambio.
- P10 (contacto válido): HTTP 200. Se creó un proveedor y un contacto, como se esperaba.
- U05 (actualización de contacto por Id sin `Proposito`): HTTP 200. La actualización parcial sigue permitida.
- Tras restaurar, los conteos volvieron a proveedor 2, natural 1, jurídico 1, contacto 1, cuenta 1.

## Notas de especificación

Estado de la especificación: sin cambios. `openspec/specs/proveedor-write/spec.md` no se modifica desde estas pruebas (3.10).

Conflicto: la especificación no define qué campos requeridos debe traer un hijo al crearse. El código lo define en una lista escrita a mano.

Texto propuesto para `proveedor-write`, requisito de validación de hijos:

```
### Requirement: Campos requeridos de hijos al crear
El sistema SHALL rechazar la creación de un hijo que no trae sus campos requeridos, con 400 y la ruta del campo faltante. Una actualización de un hijo identificado por Id no exige sus campos requeridos.

#### Scenario: Contacto sin Proposito
- **WHEN** se crea un contacto sin `Proposito`
- **THEN** la respuesta es 400 y el mensaje identifica `Contacto[i].Proposito`; no se escribe ninguna fila

#### Scenario: Actualización de contacto por Id
- **WHEN** se envía un contacto con `Id` existente sin `Proposito`
- **THEN** la actualización se acepta y los campos no enviados conservan su valor
```

Pendiente para la solución definitiva: la lista de requeridos debe salir del generador (sidecar), no de una lista escrita a mano en el MID. Ver `## Propuesta`.

## Decisión
Aceptado por el usuario (17/11/2026, 20:5x). Los hijos que se crean sin sus campos requeridos responden 400 con la ruta del campo. La lista de requeridos es provisional, escrita a mano en el MID. La definición definitiva sale del generador, y el texto de `proveedor-write` se aplica por separado.
