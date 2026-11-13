# Design

## Context

Ver `proposal.md` (Why). Estado actual relevante, verificado leyendo el código:

- El parser del DBML descarta el atributo `[not null]`. La expresión `reColumna` de
  `utils/generador_configuracion/internal/generador/dbml.go` captura solo nombre y
  tipo; el grupo de corchetes `[...]` es no capturante. `Columna` solo tiene `Nombre`
  y `Tipo`. El generador no puede, hoy, conocer la nulabilidad.
- Por eso existe el marcador: `services/generar_campos_nulables.py` re-parsea el DBML
  en Python para recuperar `[not null]`, emite `services/campos_nulables_gen.go`
  (mapa `camposNulables[tabla][CampoPascal]`), y `services/nulos.go` (`validarNulos`)
  consulta ese mapa resolviendo la tabla dueña por convención de nombres
  (`snakeDeClave`), con dos `TODO` que piden mover la nulabilidad a la configuración y
  derivar la tabla de las relaciones.
- La tabla de campos tipada del cargador (`config.CampoObjeto`) ya lleva `Readable`,
  `Writable`, `Immutable` y un `Required` que el generador nunca puebla. El
  `config.CampoConfig` del generador refleja la misma forma.
- El cargador proyecta esos roles a `WriteConfig` como listas de nombres:
  `aplicarRaiz` toma `obj.Campos.CamposEscritura()` y `CamposInmutables()`, y la
  composición por relación hace lo propio. `WriteConfig` no transporta tipos ni
  nulabilidad; solo listas planas de nombres.
- El motor de escritura (`helper/write_engine.go`) valida el cuerpo nivel por nivel en
  `validarNivel`, con `WriteConfig` y las `RelationConfig` del nivel en mano, antes de
  cualquier lectura o escritura (`Write` llama a `validarNivel` justo tras
  `desaplanarEspecializaciones` y antes de `prepass`). Es el mismo recorrido
  cuerpo-con-configuración que `validarNulos` reinventa por convención de nombres.

## Goals / Non-Goals

**Goals:**

- La nulabilidad de cada campo se deriva del DBML y viaja por el mismo canal que los
  demás roles: parser -> `CampoConfig` -> configuración generada -> `CampoObjeto` ->
  proyección a `WriteConfig` -> motor.
- El motor rechaza `null` en un escalar no nulable en `validarNivel`, usando la
  nulabilidad del árbol efectivo, antes de materializar.
- Se eliminan el script, el mapa generado y la validación por convención de nombres,
  sin cambio de contrato público.

**Non-Goals:**

- Derivar o aplicar `required` (obligación del cliente). El DBML no lo expresa (una
  columna `[not null]` con `default` no es de envío obligatorio). `required` queda sin
  poblar; si se necesita en el futuro, es un cambio aparte dirigido por un sidecar como
  `field_roles`.
- Cambiar el contrato público o la semántica observable de `null` en la escritura
  (requisito de `proveedor-write`): el resultado `400` sin escrituras se conserva.
- Redefinir la validación de inmutables, de política o de campos derivados: solo se
  añade el chequeo de nulabilidad en el mismo punto previo.

## Decisions

### Decisión 1: La nulabilidad y `required` son ejes independientes; solo se deriva la nulabilidad
`[not null]` es un hecho de almacenamiento (la columna rechaza `null`), derivable del
DBML. `required` es una obligación del cliente (debe enviarlo al escribir), que el DBML
no expresa: `activo boolean [not null, default: true]` es no nulable pero no de envío
obligatorio. Son dos hechos distintos, y el no nulable es el conjunto más amplio.

El generador deriva solo la nulabilidad, como `nullable = NOT [not null]`. No se
introduce ninguna invariante que relacione `required` y `nullable`: al mantenerlos
ortogonales, el generador no tiene que resolver una pregunta que el DBML no puede
responder, y `required` permanece sin poblar sin crear incoherencias.

Por qué: acoplar los dos ejes obligaría a inventar una regla o un sidecar para separar
los casos en que divergen (toda columna `[not null]` con `default`). Separarlos disuelve
esa pregunta y reduce el alcance: este cambio toca `nullable` y nada más.

Alternativa considerada: tratar `nullable` como el inverso de `required` y empezar a
poblar `required`. Rechazada: arrastra la derivación e imposición de `required` —un hecho
que el DBML no declara— al alcance de este cambio, con su propia lógica y pruebas, sin
necesidad.

### Decisión 2: El parser del DBML retiene `[not null]`; la nulabilidad nace en `Columna`
`reColumna` pasa a capturar el contenido del corchete de atributos, y `Columna` gana la
información de no-nulabilidad (por ejemplo `NoNula bool`, verdadera cuando los atributos
contienen `not null` o la columna es llave primaria). `construirCampos` emite
`CampoConfig.Nullable = !c.NoNula`.

Por qué: la nulabilidad es un hecho estructural del DBML, como el tipo y el nombre; su
lugar natural es el parser, no una segunda pasada. Recuperarla en el parser elimina la
razón de existir del script en Python.

### Decisión 3: La nulabilidad se emite como una bandera de rol por campo, no como un sidecar
`CampoConfig` y `CampoObjeto` ganan `Nullable bool`, emitido y cargado junto a
`readable`/`writable`/`immutable`. No se añade un sidecar de intención: a diferencia de
`field_roles`, la nulabilidad no es intención que el DBML calle, sino un hecho que el
DBML ya declara.

Por qué: mantener la nulabilidad en el mismo renglón del campo conserva una sola fuente
de verdad por campo y no amplía el vocabulario de sidecars con algo que el DBML expresa
directamente.

### Decisión 4: La nulabilidad llega al motor por una proyección de `WriteConfig`, como los demás roles
`WriteConfig` gana una lista de campos no nulables (por ejemplo `NonNullableFields`),
poblada por el cargador con una proyección de la tabla de campos
(`CamposNoNulables()`), junto a `CamposEscritura()` y `CamposInmutables()`, tanto en
`aplicarRaiz` como en la composición por relación.

Por qué: `validarNivel` recibe `WriteConfig`, no la tabla de campos tipada. La
nulabilidad debe viajar por el mismo canal que ya transporta los roles de escritura, de
modo que el motor la tenga en cada nivel sin un nuevo parámetro ni un nuevo acceso al
catálogo.

Alternativa considerada: pasar la tabla de campos completa a `validarNivel`. Rechazada:
amplía la superficie del motor y rompe la simetría con las listas de nombres que ya usa
para escribibles e inmutables.

### Decisión 5: El rechazo de `null` se hace en `validarNivel`, reemplazando a `validarNulos`
El chequeo de nulabilidad entra en `validarNivel`: para una clave cuyo valor es `null`
explícito, que está en los campos no nulables y que no es una relación, ni una FK
derivada reconocida, ni `Id`, el motor devuelve `*ErrorEscritura` (que el llamador mapea
a `400`). Se ejecuta antes de `prepass`, de modo que no deja escrituras parciales. Se
eliminan `validarNulos`, `camposNulables`, `snakeDeClave` y la llamada en
`services/proveedor_write.go`.

Por qué: `validarNivel` ya recorre el cuerpo emparejado con la configuración del nivel
y ya trata `null` de FK derivadas como ausente. Colocar aquí el chequeo honra la
decisión de `contrato-respuesta-fallo` de ubicar la distinción ausencia/null en el
motor, y elimina la resolución de tabla por convención de nombres (el segundo `TODO`),
porque el motor resuelve el objeto de cada entidad anidada por el árbol efectivo, no por
el nombre de la clave.

### Decisión 6: La configuración generada se regenera y se versiona
Añadir `nullable` a cada campo cambia todos los archivos de `config/` generados. Se
regenera con la herramienta y la salida regenerada se versiona como parte del cambio,
coherente con la generación determinista existente.

Por qué: el MID nunca interpreta el DBML en tiempo de ejecución; la salida generada es
el contrato que el cargador consume. La nulabilidad solo surte efecto cuando está en los
archivos generados.

## Risks / Trade-offs

- [La verdad regenerada puede diferir del mapa marcador en campos concretos de
  auditoría] → El marcador lista la nulabilidad de ciertos campos (por ejemplo
  `FechaModificacion`) según su propio parseo; la verdad derivada del DBML por el
  generador es la autoridad. Las pruebas de `test/nulos_test.go` fijan el
  comportamiento observable y son el árbitro de cualquier divergencia.
- [La captura del atributo en `reColumna` podría alterar el parseo de otras columnas]
  → El cambio del parser se cubre con pruebas de que una columna con `[not null]`, una
  sin atributos y una con otros atributos se parsean con la nulabilidad correcta y sin
  regresión en nombre ni tipo, antes de tocar la construcción de campos.
- [La proyección `NonNullableFields` debe poblarse en todos los puntos de composición]
  → Tanto `aplicarRaiz` como la composición por relación proyectan la nueva lista; una
  prueba de carga verifica que un objeto no raíz anidado también la lleva, no solo la
  raíz.
- [El motor debe excluir del chequeo las FK derivadas y las relaciones] → El chequeo
  reutiliza las mismas exclusiones que `validarNivel` ya aplica a `null` (relaciones,
  FK derivadas reconocidas, `Id`), de modo que un `null` en una FK relacional sigue
  tratándose como ausente y no se rechaza.

## Migration Plan

- Parser y generación: `Columna` retiene la no-nulabilidad; `CampoConfig` emite
  `nullable`; el README del generador documenta la bandera. Se regenera `config/` y la
  salida se versiona.
- Cargador: `CampoObjeto` acepta `nullable`; se añade la proyección `CamposNoNulables()`
  y el campo `NonNullableFields` de `WriteConfig`, poblado en `aplicarRaiz` y en la
  composición por relación.
- Motor: `validarNivel` rechaza `null` en un escalar no nulable con la lista proyectada.
- Retiro: se eliminan `services/generar_campos_nulables.py`,
  `services/campos_nulables_gen.go`, el camino `validarNulos`/`camposNulables`/
  `snakeDeClave` de `services/nulos.go` y la invocación en `services/proveedor_write.go`.
- Rollback: revertir el parser, la configuración regenerada, la proyección y el chequeo
  restituye el estado previo; no hay estado persistido propio del MID que deshacer.
