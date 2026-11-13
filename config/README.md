# Configuración de relaciones

Este directorio declara cómo el middleware expande (lectura) y escribe en cascada (escritura) los objetos de cada servicio. Un objeto se define una sola vez y cada servicio lo referencia con su propia arista. El cargador compone el árbol efectivo de cada servicio al arranque. Los motores de lectura y escritura reciben ese árbol ya resuelto y no resuelven referencias por sí mismos.

Hay dos tipos de archivo:

- `objetos_<nombre>.yaml`: un objeto direccionable por archivo.
- `relaciones_<servicio>.yaml`: la configuración de un servicio, con sus referencias.

## Disposición de archivos

Los archivos se agrupan en un subdirectorio por CRUD, bajo este directorio raíz. El nombre del subdirectorio es el valor de `related_api` de los objetos que contiene. El archivo `relaciones_<servicio>.yaml` va en el subdirectorio del CRUD que expone el servicio. El cargador recorre todo el árbol a cualquier profundidad, así que la carpeta sirve para organizar y no cambia la composición: mover un archivo entre subdirectorios deja el árbol efectivo igual. Los nombres de objeto siguen siendo únicos en todo el catálogo, sin importar la carpeta.

```text
config/   (extracto)
├── README.md
├── global_blacklist.yaml
├── gestion_personas_crud/
│   ├── objetos_contacto.yaml
│   ├── objetos_proveedor.yaml
│   └── relaciones_proveedor.yaml
└── terceros/
    ├── objetos_tercero.yaml
    └── objetos_tipo_contribuyente.yaml
```

## Objetos direccionables

Un objeto es una entidad con nombre único. Puede referenciarse desde varios servicios u otros objetos.

```yaml
name: Contacto
related_api: gestion_personas_crud
related_table: contacto
endpoint: /v1/contacto
campos:
  - name: NombreContacto
    type: string
    readable: true
    writable: true
referencias: []
```

| Clave | Obligatoria | Descripción |
| --- | --- | --- |
| `name` | Sí | Nombre único del objeto en el catálogo. |
| `related_api` | Sí | CRUD que almacena el objeto. Debe existir en `conf/app.conf` (`terceros`, `parametros`, `gestion_personas_crud` o `ubicaciones`). |
| `related_table` | Sí | Tabla del CRUD. |
| `endpoint` | Sí | Ruta del recurso en el CRUD. |
| `related_service` | No | Llave lógica del nodo. Si se omite, se deriva del nombre en `snake_case` (`TipoContribuyente` → `tipo_contribuyente`). Las siglas cuentan como una palabra (`HTTPServer` → `http_server`). |
| `campos` | No | Tabla de campos tipada (ver abajo). Por omisión, vacía. |
| `natural_key` | No | Campos que identifican un registro existente en la escritura idempotente. Deben ser `writable`. |
| `referencias` | No | Objetos que este objeto referencia. Por omisión, ninguno. |

### Tabla de campos tipada

Cada campo declara su tipo de almacenamiento y sus roles. Es la única fuente de verdad de los campos del objeto: la lectura usa `readable`, la escritura usa `writable` e `immutable`, y un `null` en un campo escribible con `nullable: false` se rechaza con `400`.

| Bandera | Significado |
| --- | --- |
| `type` | Obligatorio. Uno de: `boolean`, `integer`, `jsonb`, `numeric`, `string`, `timestamp`, `uuid`. |
| `nullable` | El campo admite `null` en el almacenamiento. Lo genera el generador desde el DBML: `false` cuando la columna es `[not null]` o llave primaria. Un campo sin la clave no admite `null`. Es independiente de `required`. |
| `readable` | El campo se entrega en las respuestas de lectura. |
| `writable` | El cliente puede enviar el campo en una escritura. |
| `immutable` | El campo no cambia tras crearse. Exige `writable: true`. |
| `required` | El campo es obligatorio en una escritura. Exige `writable: true`. |

El `type` describe el almacenamiento. No fija la forma en que el campo viaja en JSON ni cómo se documenta.

Las claves desconocidas en un archivo abortan la carga.

## Referencias de servicio y de objeto

Una referencia une un referenciador (un servicio o un objeto) con un objeto. La arista vive en la referencia, no en el objeto. Así un mismo objeto puede participar en servicios distintos con aristas distintas.

```yaml
service: proveedor
max_depth: 4
on_relation_error: notify
max_related_ids: 100
referencias:
  - objeto: Contacto
    direction: related_to_local
    local_field: Id
    related_field: ProveedorId
    cardinality: many
    depth: 1
    mapped_name: Contacto
    key_propagation: fk
    policy:
      creatable: true
      updatable: true
      deletable: true
```

Campos de la arista:

| Clave | Valores | Descripción |
| --- | --- | --- |
| `objeto` | Nombre de un objeto | Objeto referenciado. Obligatorio y debe existir en el catálogo. |
| `direction` | `local_to_related` o `related_to_local` | Hacia dónde apunta la llave. |
| `local_field` | Campo | Campo del referenciador que une la relación. Obligatorio. |
| `related_field` | Campo | Campo del objeto que une la relación. Obligatorio. |
| `cardinality` | `one` o `many` | Relación singular o colección. |
| `depth` | Entero positivo | Profundidad de la relación en la expansión. |
| `mapped_name` | Texto | Clave con la que la entidad aparece en la respuesta. Única entre hermanos. |
| `key_propagation` | `fk`, `shared_pk`, `fk_inbound` o vacío | Cómo se propaga la llave en la escritura. Obligatorio si `policy` admite algún verbo. |
| `policy` | `creatable`, `updatable`, `deletable` | Verbos de escritura admitidos por la arista. |

Modos de propagación:

- `fk`: el hijo guarda una llave foránea hacia el padre (`related_field`).
- `shared_pk`: el hijo comparte la llave primaria del padre.
- `fk_inbound`: el padre guarda una llave foránea hacia el objeto referenciado, que se
  escribe antes que el padre (`local_field`).

Una arista sin `key_propagation` y con `policy` vacía participa solo en lectura.

Los objetos referenciados por un servicio forman parte de su árbol efectivo. Un objeto que ningún servicio alcanza se reporta en el registro del arranque como huérfano y no entra a ningún árbol. El huérfano no aborta el arranque.

### Raíz del servicio

El campo `raiz` del archivo `relaciones_<servicio>.yaml` nombra el objeto que representa la entidad raíz del servicio. El objeto raíz se compone sin que ninguna referencia lo nombre, por lo que no se reporta como huérfano.

```yaml
service: proveedor
raiz: Proveedor
write:
  policy:
    creatable: true
    updatable: true
    deletable: false
```

Los campos escribibles, inmutables y la `natural_key` de la raíz salen de la tabla de campos de `Proveedor`. El bloque `write` solo declara `policy`. Declarar `writable_fields`, `immutable_fields` o `natural_key` en `write` junto con `raiz` aborta el arranque.

## Validación al arranque

El API no inicia si la configuración tiene alguno de estos problemas:

- Nombre de objeto duplicado, o nombre vacío.
- Campo sin `type`, con un `type` fuera del vocabulario, o repetido dentro del objeto.
- `immutable: true` o `required: true` sin `writable: true`.
- Arista inválida: `direction`, `cardinality` o `key_propagation` fuera de sus valores, `depth` no positivo, o `objeto`, `mapped_name`, `local_field` o `related_field` vacíos.
- `policy` con algún verbo admitido y sin `key_propagation`.
- `mapped_name` repetido entre referencias hermanas.
- `raiz` que no existe en el catálogo, o `write` con `writable_fields`, `immutable_fields` o `natural_key` junto con `raiz`.
- `policy` con algún verbo admitido y sin `key_propagation`.
- `mapped_name` repetido entre referencias hermanas.
- Referencia colgante: un servicio u objeto apunta a un objeto que no existe. El mensaje nombra el objeto ausente y quién lo referencia.
- Ciclo de referencias entre objetos. El mensaje muestra el camino que cierra el ciclo.
- Profundidad de composición mayor que 8 niveles. El mensaje nombra la referencia.
- `related_api` que no existe en `conf/app.conf`, o `related_table` o `endpoint` vacíos.
- `natural_key` con campos que no son `writable`.

Cada mensaje nombra el archivo, el objeto o la referencia afectados.

La validación de integridad de la composición ocurre una sola vez, al arranque. Un ciclo de datos durante una petición (por ejemplo, un registro que apunta a sí mismo a través de dos CRUD) no es un error de arranque. La expansión lo detiene por su límite de profundidad y registra un aviso `cycle` en la respuesta.
