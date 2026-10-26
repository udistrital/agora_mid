# Proposal

## Why

Los objetos anidados que entrega la expansión solo son seleccionables y
referenciables si llevan su propio `Id`, pero hoy el `Id` solo aparece cuando la
configuración lo declara en `allowed_fields` de cada relación: un olvido deja
objetos anónimos (no seleccionables). A la vez, no existe forma de suprimir de
manera transversal un campo sensible (por ejemplo un `Id` que no debe exponerse)
ni de corregir, por encima de la configuración por servicio, un campo incluido
por error. Se necesita que el `Id` esté siempre disponible por defecto y una
lista negra global que actúe como excepción y como red de seguridad sobre la
configuración de cada servicio.

## What Changes

- Se corrige la **proyección estricta de `fields`** en la expansión: hoy el motor
  inserta el registro completo que entrega el resolver, pero una selección por
  `fields` debe resolver cada ruta según su último segmento (una relación terminal
  se expande completa; un escalar terminal entrega solo ese escalar), con los
  segmentos intermedios como contenedores esqueleto del camino solicitado y sin
  las relaciones hermanas no nombradas. Esta proyección es el sustrato sobre el
  que operan el `Id` implícito y la lista negra.
- El `Id` de toda entidad expandida queda **disponible de forma implícita**: la
  configuración ya no necesita declararlo en `allowed_fields` para que una
  relación sea identificable. Cada tabla tiene `Id`, de modo que la expansión lo
  conserva aunque no esté listado.
- Una selección por `fields` que no mencione el `Id` lo omite como a cualquier
  otro escalar no solicitado (caso particular de la proyección estricta, sin
  sintaxis de exclusión); la omisión es local a la solicitud y, sin `fields`, el
  `Id` se conserva por defecto.
- Se introduce una **lista negra global** declarada en un archivo único
  `config/global_blacklist.yaml`, cargada al arranque. Una entrada se expresa de
  forma jerárquica `related_api -> related_table -> [field]`, con la palabra
  reservada `ANY` como comodín en cualquier nivel. Se prohíbe `ANY/ANY/ANY`.
- La lista negra **gana sobre** `allowed_fields` y sobre el `Id` implícito:
  cualquier campo listado se elimina de la respuesta de expansión, provenga de la
  configuración permitida o del `Id` implícito.
- La lista negra se aplica como un recorte del motor de expansión **sobre la
  salida de todo resolver** (específico y genérico por igual), de modo que el
  resolver genérico —que inserta el registro sin transformar— también la obedece
  y no filtra campos vetados.
- Pedir explícitamente en `fields` un campo vetado por la lista negra produce el
  **mismo resultado** que pedir un campo no permitido (la solicitud se rechaza),
  reutilizando el error genérico existente sin anunciar que el campo existe ni que
  está en la lista negra. En cualquier otro caso el campo vetado se omite en
  silencio, sin aviso.
- Se corrige la deriva actual de tipos de aviso: `generic_resolver` se emite pero
  no tiene constante declarada. Se establece el invariante de que **todo tipo de
  aviso emitido tiene una constante declarada** y se añade la constante
  `generic_resolver`. La lista negra **no** emite ningún aviso nuevo, por lo que
  no se agrega ningún tipo de aviso para ella.

## Capabilities

### New Capabilities
<!-- Ninguna capacidad nueva: se modifican capacidades existentes. -->

### Modified Capabilities
- `nested-object-expansion`: se refuerza la selección por `fields` con proyección
  estricta de escalares (devolver solo las rutas solicitadas, corrigiendo la
  desviación que inserta el registro completo); se añade que el `Id` de toda
  entidad expandida está disponible de forma implícita (sin declararlo en
  `allowed_fields`) y que una selección por `fields` puede omitirlo como a
  cualquier escalar; se añade la lista negra global de campos (archivo, estructura
  jerárquica con comodín `ANY`, precedencia sobre `allowed_fields` e `Id`
  implícito, aplicación sobre la salida de todo resolver, validación al arranque) y
  el resultado de pedir un campo vetado en `fields`.
- `api-response-contract`: se precisa la enumeración de tipos de aviso de
  `MetaData.Errors` para que coincida con los realmente emitidos
  (`relation`, `depth`, `cycle`, `generic_resolver`) y se declara el invariante de
  que todo tipo de aviso emitido tiene una constante declarada.

## Impact

- **APIs**: ninguna ruta nueva. Cambia la forma de la carga expandida: las
  entidades anidadas conservan su `Id` por defecto; algunos campos pueden quedar
  suprimidos globalmente por la lista negra. Pedir un campo vetado en `fields`
  tiene el mismo resultado observable que pedir un campo no permitido.
- **Configuración**: nuevo archivo `config/global_blacklist.yaml` cargado y
  validado al arranque (entradas con componentes exactos deben existir en el
  modelo conocido; un comodín `ANY` no se verifica; `ANY/ANY/ANY` aborta el
  arranque). Las `allowed_fields` de `config/relaciones_proveedor.yaml` dejan de
  necesitar `Id`.
- **Código**: motor de expansión (`helper/expand.go`) para introducir el recorte
  proyectivo de escalares sobre la salida de cada resolver (corrige la desviación
  actual que inserta el registro completo), e inyectar sobre ese mismo punto el
  `Id` implícito y aplicar el recorte de lista negra; validación/parseo de la
  selección (`helper/seleccion.go`) para que el `Id` sea seleccionable sin estar en
  `allowed_fields`; carga y validación de la lista negra en `config/`; constantes
  de tipos de aviso en `helper/expand.go` (añadir `generic_resolver`).
- **Contrato de respuesta**: `api-response-contract` precisa la enumeración de
  tipos de aviso; no se añade un tipo nuevo para la lista negra.
- **Relación con el cambio en curso `proveedor-read-post-consulta`**: son
  independientes y pueden implementarse en cualquier orden. Comparten la rutina de
  validación de `fields` y las pruebas del controlador de proveedor; conviene
  construir primero la migración de transporte (`proveedor-read-post-consulta`)
  para no rehacer dos veces esa ruta compartida. No hay conflicto de
  comportamiento: el `Id` implícito coexiste con el `Id` explícito y el rechazo de
  un campo vetado coincide con el rechazo de un campo no permitido que ese cambio
  ya estandariza.
- **`declarative-write-engine`**: sin cambios. La ausencia de `Id` en un cuerpo de
  escritura sigue significando "crear"; la regla de `Id` implícito y la lista negra
  son exclusivas de la lectura/expansión.
- **Pruebas**: proyección estricta por `fields` (solo rutas solicitadas; sin
  recorte cuando no hay `fields`); casos de `Id` implícito presente sin estar en
  `allowed_fields`; omisión de `Id` por `fields`; recorte por lista negra sobre
  resolver específico y genérico; validación de arranque de la lista negra;
  invariante de constantes de aviso.
