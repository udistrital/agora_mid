# F08: API.md usa nombres de campo que el CRUD no tiene

Estado: abierto. Documentación.

## Explicación
`API.md` (líneas 53 y 200) usa `ExperienciaLaboralMeses` y `ExperienciaProfesionalMeses`. El CRUD, la configuración y la base usan `ExperienciaLaboral` y `ExperienciaProfesional` (`config/gestion_personas_crud/objetos_proveedor_natural.yaml`).

## Propuesta
Cambiar esos nombres en `API.md` y revisar `swagger/swagger.yml`. Se puede hacer junto con F18.
