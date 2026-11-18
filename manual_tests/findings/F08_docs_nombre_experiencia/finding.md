# F08: API.md usa nombres de campo que el CRUD no tiene

Estado: aceptado. Documentación.
Solución: propuesta. Ver `## Propuesta`.
Gravedad: baja. Clasificación propuesta, ver utils/README.md.

## Explicación
`API.md` (líneas 53 y 200) usa `ExperienciaLaboralMeses` y `ExperienciaProfesionalMeses`. El CRUD, la configuración y la base usan `ExperienciaLaboral` y `ExperienciaProfesional` (`config/gestion_personas_crud/objetos_proveedor_natural.yaml`).

## Propuesta
Cambiar esos nombres en `API.md` y revisar `swagger/swagger.yml`. Se puede hacer junto con F18.

## Decisión
Aprobado por el usuario (17/11/2026, 20:20). `API.md` debe usar `ExperienciaLaboral` y `ExperienciaProfesional`, los nombres de la configuración y del CRUD. El texto de `API.md` se aplica por separado, con aprobación del texto final.
