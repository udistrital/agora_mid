# API de agora_mid

Documentación del contrato público de los endpoints de `agora_mid`. El detalle
de implementación vive aquí; el `README.md` solo cubre especificaciones
técnicas, ejecución y estado del proyecto.

## Consulta de proveedor

### Endpoint

`GET /v1/proveedor`

Permite consultar un proveedor por su identificador y solicitar la expansión de campos relacionados mediante el parámetro `fields`.

### Parámetros de consulta

| Parámetro | Obligatorio | Descripción |
|---|---|---|
| `id` | Sí | Identificador numérico positivo del proveedor. |
| `fields` | No | Lista de campos separados por comas para solicitar su expansión. |

**Ejemplo de consulta básica:**

```http
GET /v1/proveedor?id=12
```

**Ejemplo solicitando campos relacionados:**

```http
GET /v1/proveedor?id=12&fields=Id,Tercero.Id
```

Los campos solicitados deben corresponder a los permitidos por la configuración de relaciones y las reglas de expansión del servicio.

### Relaciones expandibles (subconjunto mínimo actual)

La expansión se declara en `config/relaciones_proveedor.yaml`. La configuración
vigente es un **subconjunto mínimo** del modelo de datos, pensado para avanzar a
pruebas; no cubre todavía todas las entidades del modelo extendido. Una ruta en
`fields` que no esté declarada aquí responde `400` (ver *Validación de `fields`*).

Las ramas de detalle `ProveedorNatural` y `ProveedorJuridico` son mutuamente
excluyentes: el servicio expande solo la que corresponde al tipo de registro del
proveedor (ver *Resolución del tipo de registro*) y poda la otra antes de
expandir.

| Ruta (`mapped_name`) | Cardinalidad | Llave de consulta | Campos permitidos |
|---|---|---|---|
| `Tercero` | 1-1 | `tercero.Id` (vía `proveedor.tercero_id`, único) | `Id`, `NombreCompleto`, `TipoContribuyenteId` |
| `Tercero.DatosIdentificacion` | 1-N | `datos_identificacion.TerceroId` | `Id`, `Numero`, `TipoDocumentoId`, `CiudadExpedicion` |
| `Tercero.TipoContribuyente` | 1-1 | `tipo_contribuyente.Id` | `Id`, `Nombre` |
| `ProveedorNatural` | 1-1 | `proveedor_natural.Id` (PK compartida) | `Id`, `PerfilDeclarado`, `ExperienciaLaboralMeses`, `ExperienciaProfesionalMeses` |
| `ProveedorNatural.Perfil` | 1-1 | `perfil.Id` (vía `perfil_declarado`) | `Id`, `Nombre` |
| `ProveedorJuridico` | 1-1 | `proveedor_juridico.Id` (PK compartida) | `Id`, `NombreComercial`, `MatriculaMercantil`, `FechaConstitucion` |
| `Contacto` | 1-N | `contacto.ProveedorId` | `Id`, `NombreContacto`, `Finalidad`, `TipoCanal`, `ValorEspecifico`, `EsPrincipal` |
| `CuentaBancaria` | 1-N | `cuenta_bancaria.ProveedorId` | `Id`, `EntidadBancariaId`, `TipoCuentaId`, `NumeroCuenta`, `NombreTitular`, `CiudadAperturaId`, `EsPrincipal` |

Las ciudades (`CiudadExpedicion` en `DatosIdentificacion`, `CiudadAperturaId` en
`CuentaBancaria`) se entregan como **ID plano, sin expandir**: pertenecen al CRUD
de ubicaciones y no se declaran como relación en esta fase.

Parámetros de expansión vigentes: `max_depth: 4`, `max_related_ids: 100`,
`on_relation_error: notify` (una relación que falla se omite y se reporta en
`MetaData.Errors`; el recurso principal se conserva).

**Diferido (modelo extendido, aún no expandible):** `perfil_financiero` y
`responsabilidad_fiscal_perfil`, `actividad_economica_proveedor`,
`informacion_financiera`, `representacion`, `documento`, `declaracion` y
`tipo_declaracion_proveedor`, junto con los catálogos de `parametros` y
`ubicaciones`. Estas entidades existen en el modelo de datos pero no están
declaradas en la configuración actual; se incorporarán en fases posteriores.

### Respuestas HTTP

| Código | Significado |
|---|---|
| `200` | Consulta completada sin avisos de expansión. |
| `206` | Consulta completada con avisos de relaciones que no pudieron expandirse. |
| `400` | Parámetros inválidos o campos no permitidos. |
| `404` | Proveedor no encontrado. |
| `405` | Método HTTP no permitido. |
| `502` | Error de dependencia o fallo fatal de expansión. |
| `504` | Tiempo de espera agotado durante un fallo fatal de expansión. |

Las respuestas se serializan en JSON mediante el envelope común del proyecto. Las respuestas parciales pueden incluir `MetaData.Errors` con información sobre las relaciones que no pudieron resolverse.

La definición OpenAPI del contrato está en `swagger/swagger.yml`.

### Validación de `fields`

El parámetro `fields` se valida contra la configuración declarativa **antes** de
consultar el CRUD dueño. Un `fields` inválido responde `400 Bad Request` sin
producir ninguna llamada de lectura base. Se consideran inválidas las rutas
vacías, los segmentos vacíos, los duplicados y las rutas no declaradas en
`config/relaciones_proveedor.yaml`. Ambas ramas de detalle (`ProveedorNatural` y
`ProveedorJuridico`) están declaradas, de modo que solicitar la rama que no
corresponde al tipo de registro es una ruta válida que simplemente no se
materializa, no un `400`.

### Resolución del tipo de registro

El endpoint resuelve el tipo de registro de persona natural mediante el catálogo de tipos de contribuyente, utilizando el código `P_NATURAL`. La resolución valida que exista una coincidencia única, activa y con un identificador válido.

Si el catálogo no permite resolver el tipo de registro de forma inequívoca, la inicialización del servicio falla explícitamente. El identificador no debe codificarse de forma fija ni suponerse igual a `1`.
