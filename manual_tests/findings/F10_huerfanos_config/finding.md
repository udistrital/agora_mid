# F10: objetos huérfanos en la configuración

Estado: aceptado. Informativo.
Solución: aplicada_sin_commit. Ver `## Propuesta`.
Gravedad: baja. Clasificación propuesta, ver utils/README.md.

## Explicación
Al arrancar, el loader advierte que `AreaTipo`, `GrupoInfoComplementaria` y `TipoParametro` no son referenciados por ningún servicio.

## Propuesta
Borrar esos YAML si no se usarán, o ignorar la advertencia.

## Cambio aplicado

Solución sucia sin commit (17/11/2026, 20:5x), a pedido del usuario. Las referencias faltantes se agregaron en los YAML generados, sin pasar por el generador.

**Archivos modificados:**

- `config/parametros/objetos_parametro.yaml`: `referencias` con `TipoParametro` (`TipoParametroId` a `Id`, `depth` 1).
- `config/parametros/objetos_tipo_parametro.yaml`: `referencias` con `AreaTipo` (`AreaTipoId` a `Id`, `depth` 1).
- `config/terceros/objetos_info_complementaria.yaml`: `referencias` con `GrupoInfoComplementaria` (`GrupoInfoComplementariaId` a `Id`, `depth` 1).
- `services/resolvers_nucleo.go`: resolvers de lectura para `area_tipo` y `tipo_parametro` (API `parametros`) y `grupo_info_complementaria` (API `terceros`). Sin ellos, `TestNucleoResolversCubrenElArbolDeLectura` falla.

**Verificación:** `go build ./...` y `go test -count=1 ./...` pasan en todos los paquetes.

**Verificación en vivo (17/11/2026, 20:5x)**, MID reiniciado por el tester (PID 204469):
- El arranque no registra `objeto huérfano`. Tampoco aparece ninguna línea `fatal` ni `panic`.
- Las 4 advertencias de lista negra de F04 siguen presentes, sin cambio.
- `POST /v1/proveedor/consulta {"id":1}` responde 206 con 6 avisos `depth`. Se agregaron dos, por las nuevas relaciones dentro del árbol:
  - `Tercero.InfoComplementariaTercero.InfoComplementaria.GrupoInfoComplementaria`
  - `TipoProveedorDetalle.TipoParametro`
  Ambos son cortes de hijas pedidas por la profundidad, según la regla de F31.

## Notas de especificación

Estado de la especificación: sin cambios.

Riesgo: `config/parametros/*.yaml` y `config/terceros/objetos_info_complementaria.yaml` tienen la cabecera "Generado por utils/generador_configuracion. No editar a mano". Una regeneración desde el DBML y los sidecars borrará estas referencias. La corrección definitiva debe ir en la fuente del generador (`utils/generador_configuracion/entradas/agora_modelo_extendido.dbml` o sus sidecars), no en el YAML.

Cambio de respuesta: la consulta del proveedor 1 agrega dos avisos `depth`. Eso debe considerarse al aprobar el escenario de F31 y el texto de `nested-object-expansion`.

## Decisión
Aceptado por el usuario (17/11/2026, 20:5x). Las referencias de `AreaTipo`, `TipoParametro` y `GrupoInfoComplementaria` están agregadas en la configuración, con sus resolvers de lectura, y el arranque no registra huérfanos. La corrección definitiva va en la fuente del generador, por separado, porque los YAML se regeneran.
