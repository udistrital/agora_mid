# F10: objetos huérfanos en la configuración

Estado: abierto. Informativo.
Solución: propuesta. Ver `## Propuesta`.
Gravedad: baja. Clasificación propuesta, ver utils/README.md.

## Explicación
Al arrancar, el loader advierte que `AreaTipo`, `GrupoInfoComplementaria` y `TipoParametro` no son referenciados por ningún servicio.

## Propuesta
Borrar esos YAML si no se usarán, o ignorar la advertencia.
