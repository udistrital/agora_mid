# Documentación de la suite de pruebas del subárbol del proveedor

Este documento describe las pruebas de integración y de extremo a extremo (e2e)
del middleware del proveedor, ubicadas en el paquete `test/`. Las pruebas
recorren el stack completo **cliente → controlador → servicio → motor** contra CRUD
dueños simulados, sin red real ni base de datos. Cubren tanto la lectura de
colección como la escritura del subárbol del proveedor.

## Actores

- **Cliente del API**: envía cuerpos JSON para consultar el listado o materializar
  el subárbol por `POST`/`PATCH /v1/proveedor`, y el autor de la modificación
  por la cabecera `X-Autor-Id` cuando corresponde.
- **Middleware (MID)**: los routers y controladores del proveedor, los servicios
  de lectura y escritura, y los motores genéricos cableados con la configuración
  declarativa real (`config/relaciones_proveedor.yaml`, cargada desde `../config`).
- **CRUD dueños simulados**: un único servidor HTTP de prueba (`CRUDSimulado`,
  en `fixtures.go`) que imita a Terceros CRUD y a Gestión de Personas CRUD.
  Responde `{"Data":{"Id":n}}` en `POST`/`PATCH` y `{"Data":[...]}` en `GET`,
  y captura cada petición (método, path, query, cuerpo) para las aserciones.

## Fixtures y mocks compartidos (`fixtures.go`)

- `CRUDSimulado` / `NuevoCRUDSimulado(t, existentes)`: servidor CRUD simulado;
  `existentes` precarga registros que una búsqueda por clave natural o por Id debe
  devolver (para idempotencia y continuación tras creación parcial).
- `routerProveedor(t, crud)`: arma el router real de `/v1/proveedor` y de las
  consultas del proveedor sobre los servicios reales cableados contra el CRUD
  simulado, cargando el YAML de producción. Equivale a
  `routerProveedorConBlacklist(t, crud, nil)`.
- `routerProveedorConBlacklist(t, crud, blacklist)`: como `routerProveedor` pero
  inyecta una lista negra global (`config.GlobalBlacklist`) en el servicio de
  lectura, para ejercitar el recorte y el rechazo por lista negra sobre la
  configuración real. Una `blacklist` nil equivale a "sin supresión". El router
  despacha `POST /v1/proveedor/consulta` a `Consultar` y
  `POST /v1/proveedor/consulta/lista` a `ConsultarLista`, replicando el enrutado
  de `main`.
- `ejecutar(t, router, metodo, cuerpo)` / `ejecutarConsulta(t, router, id, fields)` /
  `ejecutarListaProveedor(t, router, cuerpo)`: envían, respectivamente, una
  petición con cuerpo a `POST`/`PATCH /v1/proveedor`, una consulta puntual por
  `POST /v1/proveedor/consulta` o un listado por
  `POST /v1/proveedor/consulta/lista`, y decodifican el envoltorio de respuesta.
- `cuerpoProveedorNatural()`: cuerpo de **creación** válido de un proveedor
  natural (proveedor raíz + tercero con datos de identificación y FK del tipo de
  contribuyente + rama natural + contacto).
- `cuerpoProveedorNaturalUpdate(provID, terceroID, diID)`: cuerpo de
  **reconciliación** (`PATCH`) que identifica las entidades por su Id y omite los
  inmutables.
- `cuerpoProveedorNaturalConTerceroID(terceroID, incluir)`: cuerpo de **creación**
  de un proveedor natural nuevo que varía la FK ascendente reenviada `TerceroId`;
  `incluir=false` omite el campo (ausencia), `incluir=true` lo agrega con el valor
  dado (incluido `nil` para un `null` explícito). Ejercita la matriz de la FK
  ascendente ausente/null/valor extremo a extremo.
- `CRUDSimulado.FallarPostDe(recurso, status)`: configura el mock para que el
  `POST` de un recurso responda con un status HTTP de error (p. ej. `503`),
  simulando un CRUD dueño caído para ejercitar la política de fallo tolerado
  (`notify`) de una relación.
- Rangos de Id por entidad en el mock: proveedor `100+`, tercero `200+`,
  datos_identificacion `300+`, proveedor_natural `400+`, proveedor_juridico `500+`,
  contacto `600+`, cuenta_bancaria `700+`. Permiten afirmar de qué CRUD provino
  cada Id materializado.

## Pruebas de integración (`integration_test.go`)

### TestIntegracionListadoProveedor

- **Propósito**: verificar de extremo a extremo el listado de proveedores con filtros
  de primer nivel, expansión mediante `fields`, paginación delegada, propagación
  del total del conjunto y rechazo temprano de filtros sobre relaciones.
- **Actores**:
  - **Cliente del API**: envía `POST /v1/proveedor/consulta/lista` con el cuerpo
    JSON del listado.
  - **Middleware (MID)**: `ConsultarLista`, `ProveedorService.ListarProveedores`,
    el decodificador del envoltorio del CRUD y el motor de expansión de la
    configuración real de proveedor.
  - **CRUD dueños simulados**: `CRUDSimulado`, un servidor HTTP que representa al
    CRUD de Gestión de Personas y al CRUD de Terceros, devuelve listas precargadas
    y captura las peticiones recibidas.
- **Entradas**:
  - Estado precargado en el CRUD simulado:
    - `GET /v1/proveedor`: dos proveedores, con `Id` 101 y 102, ambos con
      `TerceroId=15` y `TipoRegistroId=1`.
    - `GET /v1/tercero`: el tercero 15 con `NombreCompleto="Ana Pérez"`.
  - Solicitud válida:
    - Verbo y ruta: `POST /v1/proveedor/consulta/lista`.
    - `query`: `[`\`TerceroId:15\``, `\`Id__in:101|102\``]`; la primera
      condición usa igualdad implícita y la segunda usa una lista `__in` separada
      por `|`.
    - `fields`: `[`\`Tercero.NombreCompleto\``].
    - Paginación: `limit=1`, `offset=1`.
  - Solicitud inválida independiente:
    - `query`: `[`\`Tercero.NombreCompleto:Ana\``], que intenta filtrar por un
      campo de relación no permitido.
- **Salidas esperadas**:
  - Solicitud válida:
    - Código HTTP `200`, envoltorio con `Success: true` y `Status: "200"`.
    - `Data` contiene un solo proveedor, porque la página solicita un elemento
      desde el desplazamiento 1.
    - Cada proveedor de la página incluye `Tercero.NombreCompleto` expandido.
    - `MetaData.Total` es `2`, tomado del total informado por el CRUD y no del
      tamaño de la página (`1`). El envoltorio puede conservar avisos
      informativos de profundidad en `MetaData.Errors` sin cambiar el estado
      exitoso del listado.
    - El CRUD de proveedores recibe una consulta GET con
      `query=TerceroId:15,Id__in:101|102`; la ventana `limit`/`offset` se aplica
      en el CRUD simulado. El MID consulta el CRUD de terceros para expandir
      `Tercero` y no realiza escrituras.
  - Solicitud inválida:
    - Código HTTP `400` con respuesta no exitosa.
    - El middleware rechaza el filtro antes de consultar el CRUD de proveedores;
      el simulador no registra una lectura del recurso raíz.

### TestIntegracionListadoProveedorPaginaMixta

- **Propósito**: verificar de extremo a extremo que un listado con proveedores
  naturales y jurídicos en la misma página expande la rama que corresponde a
  cada registro según su `TipoRegistroId`, y no la rama de un único tipo para
  toda la página.
- **Actores**:
  - **Cliente del API**: envía `POST /v1/proveedor/consulta/lista`.
  - **Middleware (MID)**: `ConsultarLista`, `ProveedorService.ListarProveedores`,
    el motor de colección y la configuración real de proveedor.
  - **CRUD dueños simulados**: `CRUDSimulado`, que sirve el listado de proveedores
    y las ramas `proveedor_natural` y `proveedor_juridico` precargadas.
- **Entradas**:
  - Estado precargado en el CRUD simulado:
    - `GET /v1/proveedor`: proveedor `Id` 101 con `TipoRegistroId=1` (natural) y
      proveedor `Id` 102 con `TipoRegistroId=2` (jurídico).
    - `GET /v1/proveedor_natural`: detalle del 101 con `PerfilDeclarado=3`.
    - `GET /v1/proveedor_juridico`: detalle del 102 con `NombreComercial="ACME"`.
  - Solicitud: `POST /v1/proveedor/consulta/lista` con `query`
    `[`\`Id__in:101|102\``]`, `fields`
    `[`\`ProveedorNatural.PerfilDeclarado\``, `\`ProveedorJuridico.NombreComercial\``]`,
    `limit=10` y `offset=0`.
- **Salidas esperadas**:
  - Código HTTP `200` con `Success: true` y `Status: "200"`.
  - `Data` contiene dos proveedores, en el orden de la página.
  - El proveedor natural expande `ProveedorNatural` (con `PerfilDeclarado=3`) y
    no contiene `ProveedorJuridico`.
  - El proveedor jurídico expande `ProveedorJuridico` (con
    `NombreComercial="ACME"`) y no contiene `ProveedorNatural`.
  - `MetaData.Total` es `2`.

### TestIntegracionCreacionCompleta

- **Propósito**: verificar que un `POST /v1/proveedor` materializa el subárbol
  completo en cascada, respeta la propagación de llave `fk_inbound` del tercero y
  **canoniza** la FK ascendente reenviada desactualizada en una entidad nueva.
- **Actores**: cliente del API, middleware, CRUD simulados (sin precarga).
- **Entradas**: `POST`, cuerpo `cuerpoProveedorNatural()` (reenvía
  `TerceroId=25`), cabecera `X-Autor-Id: 9`.
- **Salidas esperadas**:
  - `206` con envoltorio `Success: true`, `Status: "206"`, porque la FK
    ascendente reenviada (`TerceroId=25`) difiere del Id autoritativo del tercero
    recién materializado: una entidad nueva canoniza y avisa.
  - `MetaData.Errors` incluye una entrada con `Kind: field_normalization`.
  - El proveedor raíz materializa `Id=101`.
  - El tercero (FK ascendente) materializa su propio `Id=201`, distinto del
    proveedor, y el proveedor lo referencia por `TerceroId=201` (valor
    autoritativo canonizado) como **valor numérico** (no cadena de texto).
  - El `ProveedorNatural` (PK compartida) no tiene Id propio: hereda el del
    proveedor como **valor numérico** (`ProveedorNatural.Id == 101`).
  - Cada `Contacto` (FK descendente) materializa su propio `Id` (rango 600) y
    referencia al proveedor por `ProveedorId=101` como **valor numérico** (no
    cadena de texto).
  - El tercero se crea **antes** que el proveedor (orden `fk_inbound`).
  - Un único `POST /v1/proveedor/`.
  - La rama jurídica, podada para un proveedor natural, nunca se materializa.
  - El autor `9` se inyecta en los `POST` desde la cabecera, no desde el cuerpo.

### TestIntegracionReconciliacionPatchIdempotente

- **Propósito**: verificar que un `PATCH` reconcilia sin duplicar cuando el
  proveedor ya existe por su `tercero_id`.
- **Actores**: cliente del API, middleware, CRUD simulados con el proveedor
  precargado (`GET /v1/proveedor` → `{Id:101, TerceroId:25}`).
- **Entradas**: `PATCH`, cuerpo `cuerpoProveedorNaturalUpdate(101, 201, 301)`.
- **Salidas esperadas**:
  - `200`; `Data.Id == 101` (se reutiliza el proveedor existente).
  - Cero `POST` de proveedor y exactamente un `PATCH` de proveedor.

### TestIntegracionContinuacionTrasCreacionParcial

- **Propósito**: verificar la continuación tras una creación parcial: lo ya
  creado se reutiliza y solo se crea lo faltante.
- **Actores**: cliente del API, middleware, CRUD simulados con proveedor
  (`Id:101`) y datos_identificacion (`Id:301`) precargados.
- **Entradas**: `PATCH`, cuerpo `cuerpoProveedorNaturalUpdate(101, 201, 301)`.
- **Salidas esperadas**:
  - `200`; `Data.Id == 101`.
  - Proveedor: 0 `POST`, 1 `PATCH`.
  - datos_identificacion existente: 0 `POST`, 1 `PATCH` (no se duplica).
  - Contacto faltante: 1 `POST` (lo ausente sí se crea).

### TestIntegracionFieldVetadoSeRechazaComoNoPermitido

- **Propósito**: verificar la **negación tipo `allowed_fields` por lista negra**
  en la ruta de **lectura por cuerpo** (`POST /v1/proveedor/consulta`): pedir en
  `fields` un campo que sí está declarado en la configuración real
  (`allowed_fields` de la relación) pero que la lista negra global veta se rechaza
  igual que un campo no permitido, sin distinguir el caso (política anti-sondeo) y
  sin consultar el CRUD dueño (fallo temprano).
- **Actores**: cliente del API (`POST /v1/proveedor/consulta` con cuerpo
  `{id, fields}`), middleware con una lista negra global inyectada, CRUD simulados
  (sin precarga).
- **Entradas**:
  - Lista negra: `terceros → tercero → [NombreCompleto]` (veta un campo que el
    YAML real sí lista en `allowed_fields` de `Tercero`).
  - Caso A: `POST /v1/proveedor/consulta` con `{id:101, fields:["Tercero.NombreCompleto"]}`
    sobre el router con la lista negra inyectada (`routerProveedorConBlacklist`).
  - Caso B (contraste): `POST /v1/proveedor/consulta` con
    `{id:101, fields:["Tercero.CampoInexistente"]}` sobre el router sin lista negra
    (`routerProveedor`), un campo realmente no permitido.
- **Salidas esperadas**:
  - Ambos casos: `400` con envoltorio `Success: false`, `Status: "400"`.
  - El mensaje del campo vetado usa la misma plantilla "no está permitido" que el
    no permitido y cita la ruta solicitada (`Tercero.NombreCompleto`), sin revelar
    que el campo existe ni que está vetado.
  - Ninguna petición al CRUD de `proveedor`: la ruta vetada se rechaza antes de
    consultar el CRUD dueño.

### TestIntegracionFKAscendenteAusente200

- **Propósito**: verificar que, con la FK ascendente (`TerceroId`) **ausente** en
  un proveedor nuevo, el motor la completa con el Id autoritativo del tercero y
  responde `200` sin aviso.
- **Actores**: cliente del API, middleware, CRUD simulados (sin precarga).
- **Entradas**: `POST`, cuerpo `cuerpoProveedorNaturalConTerceroID(nil, false)`
  (sin `TerceroId`).
- **Salidas esperadas**:
  - `200`; `MetaData.Errors` ausente o vacío (completar una FK ausente no genera
    aviso).
  - `Data.TerceroId` se completa, como **valor numérico**, con el Id del tercero
    materializado.

### TestIntegracionFKAscendenteNull200

- **Propósito**: verificar que una FK ascendente reenviada como `null` se trata
  como ausencia, se completa con el Id autoritativo y responde `200` sin aviso.
- **Actores**: cliente del API, middleware, CRUD simulados (sin precarga).
- **Entradas**: `POST`, cuerpo `cuerpoProveedorNaturalConTerceroID(nil, true)`
  (`TerceroId: null`).
- **Salidas esperadas**:
  - `200`; sin avisos.
  - `Data.TerceroId` completado numérico con el Id del tercero materializado.

### TestIntegracionReenvioDataCoincide200

- **Propósito**: verificar el reenvío **idempotente** del `Data` materializado:
  reenviar por `PATCH` el subárbol con todas sus llaves relacionales, cuando la FK
  ascendente coincide con el Id autoritativo, responde `200` sin aviso y reutiliza
  las entidades por Id sin duplicarlas.
- **Actores**: cliente del API, middleware, CRUD simulados con el proveedor
  precargado (`GET /v1/proveedor` → `{Id:101, TerceroId:201}`).
- **Entradas**: `PATCH`, cuerpo que reenvía proveedor y tercero con Id propio y
  `TerceroId == tercero.Id == 201`.
- **Salidas esperadas**:
  - `200`; `MetaData.Errors` ausente o vacío (reenvío coincidente, sin aviso).
  - `Data.TerceroId == 201` como **valor numérico**.
  - Idempotencia: 0 `POST` de proveedor y 1 `PATCH` (se reutiliza, no se
    duplica).

### TestIntegracionConflictoFKAscendenteExistente400

- **Propósito**: verificar que una FK ascendente que **difiere** en una entidad
  existente (proveedor con Id propio) es un conflicto `fk_inbound`: se rechaza con
  `400` sin reasignar ni materializar la operación incompatible.
- **Actores**: cliente del API, middleware, CRUD simulados con el proveedor
  precargado (`{Id:101, TerceroId:201}`).
- **Entradas**: `PATCH`, cuerpo con proveedor `Id:101` y `TerceroId=999` divergente
  del tercero referenciado (`Id:201`).
- **Salidas esperadas**:
  - `400` con envoltorio `Success: false`, `Status: "400"`.
  - No se materializa el proveedor en conflicto: 0 `POST` y 0 `PATCH` de
    proveedor (sin reasignación silenciosa).

### TestIntegracionConflictoFKAscendenteNoDejaEscriturasParciales

- **Propósito**: verificar el invariante **"validar antes de escribir"** para
  `fk_inbound`: un conflicto de FK ascendente en un proveedor existente responde
  `400` sin dejar **escrituras parciales** en ningún CRUD dueño. A diferencia de
  `TestIntegracionConflictoFKAscendenteExistente400` (que solo verifica el
  proveedor), esta prueba afirma además que la entidad referenciada (`tercero`) no
  recibe ninguna escritura: el motor recupera su Id autoritativo por lectura y
  concilia la FK local **antes** de materializar.
- **Actores**: cliente del API, middleware, CRUD simulados con el proveedor
  precargado (`GET /v1/proveedor` → `{Id:101, TerceroId:201}`) y el tercero
  precargado (`GET /v1/tercero` → `{Id:201, NombreCompleto:"Ana Pérez"}`).
- **Entradas**: `PATCH`, cuerpo con proveedor `Id:101` y `TerceroId=999` divergente
  del tercero referenciado (`Id:201`).
- **Salidas esperadas**:
  - `400` con envoltorio `Success: false`, `Status: "400"`.
  - Cero escrituras en `tercero`: 0 `POST` y 0 `PATCH` (solo `GET` permitidos).
  - Cero escrituras en `proveedor`: 0 `POST` y 0 `PATCH` (solo `GET` permitidos).

### TestIntegracionFalloRelacionToleradaNotify206

- **Propósito**: verificar la política de fallo **tolerado** (`notify`): cuando el
  CRUD de una relación no bloqueante responde error, el motor conserva lo ya
  materializado y responde `206` con un aviso `Kind: relation`, sin exponer el
  error interno del CRUD.
- **Actores**: cliente del API, middleware, CRUD simulados con el `POST` de
  `contacto` forzado a `503` (`FallarPostDe("contacto", 503)`).
- **Entradas**: `POST`, cuerpo `cuerpoProveedorNaturalConTerceroID(nil, false)`
  (FK ascendente ausente, para aislar el aviso de relación del de normalización).
- **Salidas esperadas**:
  - `206`; `MetaData.Errors` incluye una entrada con `Kind: relation`.
  - El aviso no expone el cuerpo interno del CRUD.
  - El proveedor sí se materializa pese al fallo del contacto (creación parcial):
    1 `POST` de proveedor.

## Pruebas de integración de cascada del subárbol (`cascada_proveedor_test.go`)

Estas pruebas ejercitan la materialización en cascada de ramas adicionales del
subárbol sobre la configuración segmentada real (los objetos declarados en
`config/objetos_*.yaml` compuestos por `config/relaciones_proveedor.yaml`): la
rama jurídica como PK compartida y la colección anidada de cuentas bancarias
como FK descendente.

### TestIntegracionCreacionJuridicoMaterializaSoloRamaJuridica

- **Propósito**: verificar que un `POST /v1/proveedor` de un registro de tipo
  **jurídico** (`TipoRegistroId=2`) materializa únicamente la rama
  `ProveedorJuridico` (PK compartida con el proveedor) y poda la rama natural: no
  crea `ProveedorNatural`.
- **Actores**: cliente del API, middleware, CRUD simulados (sin precarga).
- **Entradas**: `POST`, cuerpo con `TipoRegistroId=2`, `DescripcionPortafolio`,
  `Tercero` (`NombreCompleto="Empresa Demo SAS"`) y `ProveedorJuridico`
  (`NombreComercial`, `MatriculaMercantil`, `FechaConstitucion`, `CotizaBolsa`).
- **Salidas esperadas**:
  - Código HTTP `200`.
  - `Data.ProveedorJuridico` está presente y hereda el Id del proveedor como
    **valor numérico** (`ProveedorJuridico.Id == 101`), por ser PK compartida.
  - `Data` no contiene `ProveedorNatural`.
  - Cero `POST` a `proveedor_natural` (la rama natural, podada, nunca se
    materializa).
  - Exactamente una escritura (`POST`/`PATCH`) a `proveedor_juridico`.

### TestIntegracionCreacionCuentaBancariaReferenciaProveedorPorFK

- **Propósito**: verificar que una `CuentaBancaria` anidada (colección, FK
  descendente) se materializa con su propio Id y referencia al proveedor por
  `ProveedorId`, preservando el **tipo numérico** de la FK tanto en la respuesta
  como en el cuerpo enviado al CRUD dueño.
- **Actores**: cliente del API, middleware, CRUD simulados (sin precarga).
- **Entradas**: `POST`, cuerpo de un proveedor natural con `ProveedorNatural`
  (`PerfilDeclarado=3`) y una `CuentaBancaria` con `EntidadBancariaId`,
  `TipoCuentaId`, `NumeroCuenta="001-123456"`, `NombreTitular`, `CiudadAperturaId`
  y `EsPrincipal`.
- **Salidas esperadas**:
  - Código HTTP `200`.
  - `Data.CuentaBancaria` contiene exactamente un elemento que materializa su
    propio `Id=701` (rango de `cuenta_bancaria`) y referencia al proveedor por
    `ProveedorId=101` como **valor numérico**.
  - El `POST` de `cuenta_bancaria` recibido por el CRUD dueño envía
    `ProveedorId=101` numérico y conserva los escalares propios (p. ej.
    `NumeroCuenta="001-123456"`).

## Pruebas de integración de lectura del subárbol (`lectura_proveedor_test.go`)

### TestIntegracionConsultaSinFieldsExpandeConIdImplicitoYPodaRamaJuridica

- **Propósito**: verificar que una consulta por cuerpo **sin `fields`**
  (`POST /v1/proveedor/consulta`) expande según la configuración real: cada
  entidad anidada conserva su `Id` implícito, los campos legibles se entregan y la
  rama jurídica se poda para un proveedor natural. Los únicos avisos admisibles
  son de profundidad (`Kind: depth`).
- **Actores**: cliente del API (`POST /v1/proveedor/consulta` con `{id}`),
  middleware (lectura y motor de expansión de la configuración real), CRUD
  simulados con el subárbol natural precargado.
- **Entradas**:
  - Estado precargado en el CRUD simulado:
    - `GET /v1/proveedor`: proveedor `Id=101`, `TerceroId=201`,
      `TipoRegistroId=1` (natural), `DescripcionPortafolio="Servicios"`.
    - `GET /v1/tercero`: tercero `Id=201`, `NombreCompleto="Ana Pérez"`,
      `TipoContribuyenteId=2`.
    - `GET /v1/proveedor_natural`: detalle `Id=101`, `PerfilDeclarado=3`,
      `ExperienciaLaboralMeses=12`.
  - Solicitud: `POST /v1/proveedor/consulta` con cuerpo `{"id":101}` y cabecera
    `X-Autor-Id: 9`.
- **Salidas esperadas**:
  - Código HTTP `200` o `206`; si hay avisos en `MetaData.Errors`, todos son de
    `Kind: depth`.
  - `Data.Tercero` se expande con su `Id` implícito `201` y entrega
    `NombreCompleto="Ana Pérez"`.
  - `Data.ProveedorNatural` se expande con su `Id` implícito `101` y entrega
    `ExperienciaLaboralMeses=12`.
  - `Data` no contiene `ProveedorJuridico` (rama jurídica podada para un
    proveedor natural).

## Pruebas de integración de selección y lista negra en lectura (`seleccion_blacklist_proveedor_test.go`)

Estas pruebas ejercitan la **selección por `fields`** y la **supresión silenciosa
por lista negra** en la ruta de lectura por cuerpo (`POST /v1/proveedor/consulta`).

- `consultaProveedorCuerpo(t, router, cuerpo)`: helper local que envía
  `POST /v1/proveedor/consulta` con el cuerpo dado y la cabecera `X-Autor-Id: 9`,
  decodifica el envoltorio, acepta solo `200` o `206` y, en `206`, exige que todo
  aviso en `MetaData.Errors` sea de `Kind: depth`.

### TestIntegracionConsultaFieldsEscalarOmiteIdYHermanas

- **Propósito**: verificar que una selección por `fields` que pide **solo un
  escalar** de una relación entrega la entidad anidada **sin su `Id`** (porque la
  selección no lo menciona) y no incluye relaciones hermanas no solicitadas.
- **Actores**: cliente del API, middleware (selección por `fields` sobre la
  configuración real), CRUD simulados con el subárbol natural precargado
  (proveedor `101`, tercero `201`, proveedor_natural `101`).
- **Entradas**: `POST /v1/proveedor/consulta` con cuerpo
  `{"id":101,"fields":["Tercero.NombreCompleto"]}`.
- **Salidas esperadas**:
  - `Data.Tercero` entrega `NombreCompleto="Ana Pérez"` y **no** incluye la clave
    `Id` (la selección escalar no la menciona).
  - `Data` no contiene `ProveedorNatural` (la relación hermana no solicitada no
    aparece).

### TestIntegracionConsultaListaNegraSuprimeCampoVetado

- **Propósito**: verificar que la lista negra global, aplicada sobre una
  **expansión por configuración** (consulta sin `fields`), suprime el campo
  vetado de la entidad expandida pero conserva su `Id` implícito, y que la
  supresión es **silenciosa** (sin aviso en `MetaData`). Contrasta con el rechazo
  explícito cuando el campo vetado se pide en `fields`
  (`TestIntegracionFieldVetadoSeRechazaComoNoPermitido`).
- **Actores**: cliente del API, middleware con lista negra global inyectada
  (`routerProveedorConBlacklist`), CRUD simulados con proveedor `101` y tercero
  `201` precargados.
- **Entradas**:
  - Lista negra: `terceros → tercero → [NombreCompleto]`.
  - Solicitud: `POST /v1/proveedor/consulta` con cuerpo `{"id":101}` (expansión
    por configuración, sin `fields`).
- **Salidas esperadas**:
  - `Data.Tercero` se expande pero **no** incluye `NombreCompleto` (campo vetado
    suprimido) y conserva su `Id` implícito `201`.
  - `MetaData` no contiene avisos de `Kind: field_blacklisted` ni mensajes que
    mencionen la lista negra: la supresión es silenciosa.

## Prueba e2e (`e2e_test.go`)

### TestE2EProveedorCreacionYActualizacion

- **Propósito**: ejercitar un flujo e2e completo del subárbol sobre un único
  router y CRUD simulado: crear (`POST`) y luego actualizar (`PATCH`) reutilizando
  los identificadores materializados que devolvió la creación, como lo haría un
  cliente real que continúa sobre lo ya creado.
- **Actores**: cliente del API, middleware, CRUD dueños simulados.
- **Entradas**:
  - Fase 1: `POST`, cuerpo `cuerpoProveedorNatural()`, `X-Autor-Id: 9`.
  - Fase 2: `PATCH`, cuerpo `cuerpoProveedorNaturalUpdate(provID, terceroID,
    diID)` con los Ids devueltos por la fase 1.
- **Salidas esperadas**:
  - Fase 1: `206` (la FK ascendente reenviada `TerceroId=25` se canoniza en la
    entidad nueva, con aviso `field_normalization`); el proveedor y el tercero
    materializan Ids propios distintos; `proveedor.TerceroId == tercero.Id` (FK
    ascendente) como **valor numérico** (no cadena de texto); datos_identificacion
    materializa su Id.
  - Fase 2: `200`; opera sobre el mismo proveedor (`Data.Id == provID`).
  - Conteos a lo largo de las dos fases, por entidad con Id estable: proveedor,
    tercero y datos_identificacion con exactamente 1 `POST` (crear) y 1 `PATCH`
    (actualizar), sin duplicados.

## Matriz de comportamiento de campos relacionales derivados

Cada fila es un caso de la conciliación de un campo derivado (FK descendente,
PK compartida o FK ascendente) y la respuesta observable. La columna de cada
nivel indica qué prueba lo cubre: **unitaria** en `helper/write_engine_test.go`
(motor, con `resolverFalso`), **controlador** en
`controllers/proveedor_test.go` (mapeo a envoltorio y código HTTP) e
**integración/e2e** en `test/` (observable extremo a extremo vía router y CRUD
simulado).

| Caso | Resultado | Unitaria (`helper/`) | Controlador | Integración/e2e (`test/`) |
| --- | --- | --- | --- | --- |
| Campo derivado **ausente** | completa con la llave autoritativa, `200` sin aviso | `TestWriteFKDerivadoAusente`, `TestWriteSharedPKAusente`, `TestWriteInboundAusente` | — | `TestIntegracionFKAscendenteAusente200` |
| Campo derivado **`null`** | se trata como ausente, completa, `200` sin aviso | `TestWriteFKDerivadoNull`, `TestWriteInboundNull` | — | `TestIntegracionFKAscendenteNull200` |
| Campo derivado **coincide** | conserva, `200` sin aviso | `TestWriteFKDerivadoCoincide`, `TestWriteSharedPKCoincide`, `TestWriteInboundCoincide` | `campo derivado coincidente 200 sin aviso` (en `TestProveedorHandlerEscribir`) | `TestIntegracionReenvioDataCoincide200` |
| **Diferencia en entidad nueva** | canoniza con la llave autoritativa, `206` + `Kind: field_normalization` | `TestWriteFKDerivadoDiferenteNuevo`, `TestWriteInboundDiferenteNuevo` | `canonización de campo derivado 206 field_normalization` (en `TestProveedorHandlerEscribir`) | `TestIntegracionCreacionCompleta`, e2e fase 1 |
| **Diferencia en entidad existente** | conflicto, `400`, sin reasignar | `TestWriteFKDerivadoDiferenteExistente`, `TestWriteSharedPKDiferenteExistente`, `TestWriteInboundDiferenteExistente` | `campo inválido 400` (en `TestProveedorHandlerEscribir`) | `TestIntegracionConflictoFKAscendenteExistente400`, `TestIntegracionConflictoFKAscendenteNoDejaEscriturasParciales` |
| **Fallo de relación tolerado** (`notify`) | conserva lo materializado, `206` + `Kind: relation` | `TestWriteTolerante` | `materialización parcial 206` (en `TestProveedorHandlerEscribir`) | `TestIntegracionFalloRelacionToleradaNotify206` |
| **Fallo de relación bloqueante** (`fail`) | aborta, `502`/`504` | `TestWriteEstricto` | `fallo bloqueante 502`, `fallo bloqueante por timeout 504` (en `TestProveedorHandlerEscribir`) | — (cubierto por unitaria del motor y prueba del controlador) |

Las normalizaciones y conflictos de campos derivados aplican a los tres modos de
propagación (`fk`, `shared_pk`, `fk_inbound`); las filas listan las pruebas
unitarias por modo donde aplica. El caso `fail` no se repite en integración: lo
cubren la prueba unitaria del motor y la del controlador que mapea el error fatal
al código HTTP.

## Cómo ejecutar

```bash
go test ./test/...
```
