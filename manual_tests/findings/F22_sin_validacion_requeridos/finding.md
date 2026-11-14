# F22: no se validan los campos requeridos de un hijo

Estado: abierto. Reproducido en la corrida 4: P11 responde 200.
Solución: propuesta. Ver `## Propuesta`.
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
