# F18: nombres de campo de API.md distintos de la configuración

Estado: abierto. Documentación.
Solución: propuesta. Ver `## Propuesta`.
Gravedad: media. Clasificación propuesta, ver utils/README.md.

## Explicación
`API.md` usa nombres que la configuración y el CRUD no tienen:
- `TipoRegistroId` (raíz): es `TipoProveedor`.
- `Contacto.TipoCanal` y `Contacto.Finalidad`: son `TipoCanalContactoId` y `Proposito`.
- `Contacto.EsPrincipal`: no existe.
- `ExperienciaLaboralMeses`: es `ExperienciaLaboral` (ver F08).

Un cliente que siga `API.md` recibe 400 (campo no escribible) o campos obligatorios ausentes.

## Propuesta
Corregir `API.md` y sus ejemplos con la tabla de naming de `../escritura/matriz.md`.
