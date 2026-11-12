# Generador de configuración de relaciones

Produce los archivos `objetos_<tabla>.yaml` y `relaciones_<servicio>.yaml` del middleware a partir del modelo DBML y de cinco sidecars de intención. Se ejecuta a mano. El API no lo importa ni lo usa al arrancar.

## Estructura

```text
utils/generador_configuracion/
├── README.md
├── cmd/generador/          # Comando de línea: lee flags y llama a generador.Generar.
├── entradas/               # Entradas versionadas: DBML y sidecars.
│   ├── agora_modelo_extendido.dbml
│   ├── expand.yaml
│   ├── fk_inbound.yaml
│   ├── blacklist.yaml
│   ├── write.yaml
│   ├── service.yaml
│   ├── flatten.yaml
│   └── field_roles.yaml
└── internal/generador/     # Paquete: parser, sidecars, construcción y escritura.
```

## Uso

Desde la raíz del repositorio:

```shell
go run ./utils/generador_configuracion/cmd/generador -salida <carpeta_destino>
```

| Flag | Valor por omisión | Descripción |
| --- | --- | --- |
| `-salida` | (obligatoria) | Carpeta donde se escriben los objetos. Se crea `<related_api>/objetos_<tabla>.yaml` dentro. |
| `-carpeta` | `utils/generador_configuracion/entradas` | Carpeta con el DBML y los sidecars. |
| `-dbml` | `agora_modelo_extendido.dbml` | Nombre del DBML dentro de `-carpeta`. |

La salida es determinista: dos ejecuciones con las mismas entradas producen archivos idénticos byte a byte.

Antes de escribir, el generador elimina de `-salida` los archivos YAML que llevan el encabezado de generado, y después las carpetas que quedan vacías. Los archivos sin ese encabezado no se tocan. Si la construcción falla, la salida previa queda intacta.

Esquemas con CRUD: `proveedores` (`gestion_personas_crud`), `terceros`, `parametros` y `ubicaciones`. Las tablas de otros esquemas no generan objetos.

## Sidecars

| Archivo | Forma | Efecto |
| --- | --- | --- |
| `expand.yaml` | Lista de `esquema.tabla.columna` | Crea una arista expandible en el padre hacia el hijo que porta la FK. |
| `fk_inbound.yaml` | Lista de `esquema.tabla.columna` | Crea una arista en el hijo que porta la llave, con `key_propagation: fk_inbound`. |
| `blacklist.yaml` | Lista de `esquema.tabla.columna` | Omite la columna del objeto. No puede estar en `expand` ni en `fk_inbound`. |
| `write.yaml` | Mapa de `esquema.tabla` a `creatable`, `updatable`, `deletable` y `natural_key` | Copia la política a las aristas que apuntan al objeto y fija la clave natural. En la raíz de un servicio, la política va a `write.policy` del archivo de servicio. |
| `service.yaml` | Mapa de `esquema.tabla` de la raíz a `service`, `max_depth`, `on_relation_error`, `max_related_ids` y `filterable_fields` | Emite `relaciones_<service>.yaml` en el CRUD de la raíz. Las aristas de la raíz salen del objeto raíz y van a ese archivo, cuyas `referencias` las reciben. El objeto raíz queda con `referencias: []`. |
| `flatten.yaml` | Lista de `esquema.tabla.columna` de PK compartida del lado hijo (1-1 con `-`) | Marca con `flatten: true` la arista padre->hijo de esa unión, de modo que los campos del hijo se proyectan al padre. No emite arista hijo->padre. Falla si la columna no es PK compartida o si también está en `fk_inbound`. |
| `lookup.yaml` | Lista de `esquema.tabla.columna` que son FK, o mapa con `entrada`, `depth` y `mapped_name` | Crea una arista `local_to_related` de cardinalidad `one` en el objeto que porta la columna, con propagación vacía y política sin verbos, aunque `write` declare política para el objeto referenciado. Falla si la FK también está en `expand`, `fk_inbound`, `flatten` o `blacklist`, o si no es una FK del DBML. |
| `field_roles.yaml` | Mapa de `esquema.tabla.columna` a `readable`, `writable` e `immutable`. `ANY` solo es válido en el esquema y en la tabla, no en la columna | Fija los roles de cada campo. Precedencia: `esquema.tabla.columna` > `esquema.ANY.columna` > `ANY.tabla.columna` > `ANY.ANY.columna`. Sin entrada, el campo es legible y escribible. Falla si una entrada no coincide con ninguna columna, si usa `ANY` en el segmento de columna o si declara `immutable: true` con `writable: false`. |

Nombre de las aristas de `lookup` y `fk_inbound`: se deriva de la columna quitando el sufijo `_id` (`tercero_id` produce `Tercero`). `mapped_name` en una entrada de mapa sustituye ese nombre. Falla si el nombre resultante coincide con un campo del objeto que porta la arista; en ese caso declare `mapped_name`.

Si los objetos generados forman un ciclo de referencias, el generador falla e indica el camino que cierra el ciclo.

En `expand` y `fk_inbound`, una entrada es una cadena simple (profundidad por omisión, 1) o un mapa con `entrada` y `depth`, la profundidad de su arista:

```yaml
- proveedores.contacto.proveedor_id
- entrada: proveedores.proveedor.tercero_id
  depth: 2
```

Una FK que no aparece en `expand` ni en `fk_inbound` queda como identificador plano en el hijo. Si una FK aparece en `expand` y en `fk_inbound`, prevalece `fk_inbound`. Un archivo de sidecar ausente equivale a una lista vacía. Las claves desconocidas en un sidecar son error.

## Pruebas

```shell
go test ./utils/generador_configuracion/...
```
