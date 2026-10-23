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

## Escritura de proveedor

El proveedor se escribe como **recurso raíz** mediante un cuerpo JSON anidado que
representa su subárbol del núcleo. El MID coordina la cascada a través de los CRUD
dueños (terceros y gestión de personas) pero no es dueño de los datos ni maestro
de identidad.

### Endpoints

| Verbo | Ruta | Propósito |
|---|---|---|
| `POST` | `/v1/proveedor` | Crea el subárbol del proveedor en cascada. |
| `PATCH` | `/v1/proveedor` | Reconcilia idempotentemente el subárbol: crea lo ausente, actualiza lo presente y nunca borra. |

Ambos verbos comparten la ruta del recurso raíz `GET /v1/proveedor`; el enrutado
por método delega `POST`/`PATCH` en el servicio de escritura y el resto en la
lectura.

### Cabeceras

| Cabecera | Obligatoria | Descripción |
|---|---|---|
| `X-Autor-Id` | No | Identificador entero positivo del autor de la modificación, poblado por la capa de autenticación. Se inyecta en los registros materializados. |

### Cuerpo anidado

El cuerpo es un objeto JSON con el proveedor raíz y sus entidades anidadas. Solo
se admiten los **campos escribibles declarados por relación** en
`config/relaciones_proveedor.yaml` (bloque `write`); cualquier otro campo
responde `400` sin materializar nada.

| Nodo | Cardinalidad | Campos escribibles | Reglas |
|---|---|---|---|
| (raíz) `proveedor` | 1 | `TerceroId`, `TipoRegistroId`, `DescripcionPortafolio` | `TipoRegistroId` obligatorio. `TerceroId` es clave natural e **inmutable**: cambiarlo en `PATCH` responde `400`; puede omitirse y se propaga desde el `Tercero` materializado. |
| `Tercero` | 1-1 | `NombreCompleto`, `PrimerNombre`, `SegundoNombre`, `PrimerApellido`, `SegundoApellido`, `FechaNacimiento`, `TipoContribuyenteId` | Se crea/actualiza en el CRUD de terceros pero **nunca se elimina**; su `Id` se propaga a `proveedor.TerceroId`. |
| `Tercero.DatosIdentificacion` | 1-N | `Numero`, `TipoDocumentoId`, `DigitoVerificacion`, `CiudadExpedicion`, `FechaExpedicion` | Idempotencia por clave natural `Numero`+`TipoDocumentoId` (ambos **inmutables**). No se elimina. |
| `ProveedorNatural` | 1-1 | `PerfilDeclarado`, `ExperienciaLaboralMeses`, `ExperienciaProfesionalMeses` | PK compartida con `proveedor`. Válido solo si `TipoRegistroId` es persona natural. |
| `ProveedorJuridico` | 1-1 | `NombreComercial`, `MatriculaMercantil`, `FechaConstitucion`, `FechaRenovacion`, `ReportaBeneficiosFinales`, `CotizaBolsa`, `RequiereRevisorFiscal` | PK compartida con `proveedor`. Válido solo si `TipoRegistroId` es persona jurídica. |
| `Contacto` | 1-N | `NombreContacto`, `Finalidad`, `TipoCanal`, `ValorEspecifico`, `Extension`, `EsPrincipal` | Colección sin clave natural: elemento sin `Id` se crea, con `Id` se actualiza. |
| `CuentaBancaria` | 1-N | `EntidadBancariaId`, `TipoCuentaId`, `NumeroCuenta`, `NombreTitular`, `CiudadAperturaId`, `EsPrincipal` | Colección sin clave natural: elemento sin `Id` se crea, con `Id` se actualiza. |

Las ramas `ProveedorNatural` y `ProveedorJuridico` son **mutuamente
excluyentes**: el cuerpo debe traer únicamente la que corresponde a
`TipoRegistroId`; declarar la rama incorrecta, o ambas, responde `400`.

Las entidades de parámetros y catálogos (`TipoContribuyenteId`, `PerfilDeclarado`,
`EntidadBancariaId`, `TipoCuentaId`) y las ciudades (`CiudadExpedicion`,
`CiudadAperturaId`) se referencian **solo por su `Id`**: el MID nunca las crea ni
modifica.

**Ejemplo de cuerpo (`POST`, persona natural):**

```json
{
  "TipoRegistroId": 1,
  "DescripcionPortafolio": "Servicios de consultoría",
  "Tercero": {
    "NombreCompleto": "Ana Pérez",
    "TipoContribuyenteId": 1,
    "DatosIdentificacion": [
      { "Numero": "1012345678", "TipoDocumentoId": 3, "CiudadExpedicion": 149 }
    ]
  },
  "ProveedorNatural": {
    "PerfilDeclarado": 5,
    "ExperienciaProfesionalMeses": 60
  },
  "Contacto": [
    { "NombreContacto": "Ana Pérez", "TipoCanal": "correo", "ValorEspecifico": "ana@example.com", "EsPrincipal": true }
  ],
  "CuentaBancaria": [
    { "EntidadBancariaId": 7, "TipoCuentaId": 1, "NumeroCuenta": "123456789", "NombreTitular": "Ana Pérez", "EsPrincipal": true }
  ]
}
```

### Idempotencia y reconciliación (`PATCH`)

`PATCH /v1/proveedor` reconcilia el subárbol: crea lo ausente, actualiza lo
presente y nunca borra, de modo que reenviar el mismo cuerpo produce el mismo
estado final. Las entidades existentes se reutilizan por su clave natural
(`proveedor` por `TerceroId`, `DatosIdentificacion` por `Numero`+`TipoDocumentoId`)
o por el `Id` del cuerpo en las colecciones. Esto permite **continuar** un
`POST`/`PATCH` previo que dejó el subárbol parcialmente creado, sin duplicar
registros.

### Respuestas HTTP

| Código | Significado |
|---|---|
| `200` | Creación/actualización completa sin avisos. `Data` incluye los identificadores materializados; `MetaData` se omite. |
| `206` | Materialización parcial tolerada: una relación no bloqueante falló tras materializar otras. El recurso se conserva, `Data` incluye los identificadores ya materializados y los avisos van en `MetaData.Errors`. |
| `400` | Cuerpo o campo inválido: cuerpo no es un objeto JSON, `TipoRegistroId` ausente o no positivo, campo no escribible o inmutable, o rama de detalle incoherente con el tipo de registro. No se materializa nada. |
| `405` | Método HTTP no permitido; solo se admiten `POST` y `PATCH` (además de `GET`). |
| `502` | Fallo bloqueante de una relación bajo política de error estricta al escribir en un CRUD dueño. |
| `504` | Tiempo de espera agotado durante un fallo bloqueante de escritura. |

Las respuestas usan el mismo envelope plano `Success`/`Status`/`Message`/`Data`
del proyecto. `MetaData.Errors` reporta las relaciones que no pudieron
materializarse sin exponer URLs ni datos sensibles.

La definición OpenAPI de estos endpoints está en `swagger/swagger.yml`.
