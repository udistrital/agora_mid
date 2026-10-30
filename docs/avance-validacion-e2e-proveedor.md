# Avance de validación E2E - Consulta de proveedores

**Proyecto:** agora_mid_issue49
**Rama:** feature/49-add-collection-read
**Fecha:** 29 de octubre de 2026
**Endpoint:** POST /v1/proveedor/consulta/lista

## 1. Objetivo

Realizar una validación de extremo a extremo del endpoint de consulta de proveedores implementado en el MID, utilizando los servicios reales terceros_crud y gestion_personas_crud, con bases de datos de prueba y registros relacionados.

Se busca verificar:
- Consulta del proveedor desde el MID.
- Recuperación de relaciones entre servicios.
- Expansión de relaciones anidadas.
- Aplicación de filtros y paginación.
- Recuperación correcta de MetaData.Total.

## 2. Entorno de pruebas

| Componente | Puerto | Base de datos |
|---|---:|---|
| terceros_crud | 8084 | terceros_crud_test |
| gestion_personas_crud | 8086 | gestion_personas_crud_test |
| agora_mid_issue49 | 8090 | No aplica |

Los servicios CRUD se ejecutaron contra sus respectivas bases de datos de prueba. El MID se configuró para consumir ambos servicios.

Las variables de entorno utilizadas para la conexión se omitieron por seguridad.

## 3. Preparación de datos

### 3.1. Tercero

Se creó un tercero mediante la API de terceros_crud.

- ID: 1
- NombreCompleto: Proveedor de prueba E2E
- Tipo de contribuyente: Persona natural
- TipoContribuyenteId: 1
- Código de abreviación: P_NATURAL

Se verificó mediante GET /v1/tercero/1.

### 3.2. Perfil

Se creó un perfil mediante la API de gestion_personas_crud.

- ID: 1
- Nombre: Perfil de prueba E2E
- Descripción: Perfil temporal para validar agora_mid_issue49
- Código de abreviación: PERFIL_E2E
- Activo: true
- Número de orden: 99

Se verificó mediante GET /v1/perfil/1.

### 3.3. Proveedor

Se creó el registro raíz mediante POST /v1/proveedor.

- ID: 2
- TerceroId: 1
- TipoRegistroId: 1
- Activo: true

La primera inserción falló porque FechaCreacion llegó como NULL y la columna no admite valores nulos. Se resolvió enviando explícitamente FechaCreacion en formato RFC3339. La segunda petición respondió HTTP 201 Created.

El primer intento fallido consumió un valor de la secuencia, por lo que el registro creado recibió ID 2.

### 3.4. Proveedor natural

Se creó el detalle mediante POST /v1/proveedor_natural.

- ID: 2
- PerfilDeclarado: ID 1
- ExperienciaLaboralMeses: 12
- ExperienciaProfesionalMeses: 6
- Activo: true

La tabla utiliza una clave primaria compartida con proveedor. Se ajustó la secuencia de proveedor_natural para que el siguiente identificador fuera 2.

La inserción respondió HTTP 201 Created.

Se verificó mediante:
- Consulta SQL con JOIN entre proveedor y proveedor_natural.
- GET /v1/proveedor_natural/2, que respondió HTTP 200.

La consulta SQL confirmó que ambos registros tienen ID 2 y que las referencias a tercero y perfil están almacenadas correctamente.


## 4. Inicio del MID

Se inició agora_mid_issue49 en el puerto 8090 mediante go run. El log confirmó: http server Running on http://:8090.

Se configuraron las URL de terceros_crud (8084), gestion_personas_crud (8086) y parametros_crud (8087). Durante el arranque se resolvió el tipo de contribuyente P_NATURAL.

## 5. Pruebas realizadas

### 5.1. Consulta básica del listado

Se ejecutó POST /v1/proveedor/consulta/lista con query vacío, fields vacío, limit 10 y offset 0.

**Resultado: HTTP 200 OK.** Se recuperó el proveedor ID 2, con TerceroId 1 y TipoRegistroId 1. MetaData.Total devolvió 1.

**Estado: satisfactorio para la consulta básica.**

### 5.2. Consulta con expansión de relaciones

Se repitió la petición solicitando los campos Id, Activo, Tercero y ProveedorNatural.

**Resultado: HTTP 206 Partial Content.** Se recuperaron Tercero, ProveedorNatural, TipoContribuyenteId y PerfilDeclarado. Sin embargo, las relaciones anidadas Tercero.TipoContribuyente y ProveedorNatural.Perfil no se resolvieron.

También se recibieron advertencias de profundidad para Tercero, Tercero.DatosIdentificacion, Tercero.TipoContribuyente y ProveedorNatural. MetaData.Total continuó siendo 1.

**Estado: parcial.**

## 6. Análisis técnico del problema

Se inspeccionó config/relaciones_proveedor.yaml y se identificaron dos relaciones anidadas con problemas de resolución:

- Tercero.TipoContribuyente: utiliza local_field TipoContribuyenteId.
- ProveedorNatural.Perfil: utiliza local_field PerfilDeclarado.

En las respuestas de los CRUD, ambos campos llegan como objetos que contienen una propiedad Id.

En helper/expand.go se revisaron las funciones llaveLocal y aCadena. llaveLocal obtiene directamente el valor del campo local configurado y lo convierte mediante aCadena. Esta última contempla string, float64, int, int64 y bool, pero no contempla explícitamente map[string]any ni extrae la propiedad Id de un objeto.

Esto constituye una hipótesis técnica consistente con los errores observados: el expansor podría estar utilizando la representación textual del mapa en lugar del identificador interno.

La corrección todavía no se ha implementado ni probado.

## 7. Archivos y funciones inspeccionados

Archivos:
- main.go
- conf/app.conf
- config/relaciones_proveedor.yaml
- helper/expand.go

Funciones:
- expandirValor
- expandirNivel
- expandirRelacionNivel
- llaveLocal
- resolver
- filtrarPorRelatedField
- aCadena

## 8. Actividades pendientes

1. Revisar las pruebas existentes del expansor.
2. Diseñar pruebas unitarias para identificadores numéricos, cadenas, mapas con propiedad Id y referencias nulas o sin Id.
3. Implementar la corrección genérica si las pruebas confirman la hipótesis.
4. Ejecutar las pruebas unitarias y la suite del proyecto.
5. Repetir la consulta con expansión de relaciones.
6. Verificar Tercero.TipoContribuyente y ProveedorNatural.Perfil.
7. Revisar los límites de profundidad configurados.
8. Probar filtros y paginación, incluyendo MetaData.Total.
9. Documentar los resultados finales de la validación E2E.

## 9. Estado general

| Validación | Estado |
|---|---|
| Servicios CRUD operativos | Correcto |
| Datos de prueba creados | Correcto |
| Clave primaria compartida | Correcto |
| MID iniciado | Correcto |
| Consulta básica del listado | HTTP 200 |
| MetaData.Total en consulta básica | Correcto |
| Expansión de Tercero | Parcial |
| Expansión de ProveedorNatural | Parcial |
| Relaciones anidadas | Pendiente |
| Pruebas de filtros | Pendiente |
| Pruebas de paginación | Pendiente |
| Validación E2E completa | Pendiente |

## 10. Conclusión

Se comprobó que el endpoint de listado funciona mediante una petición real al MID y recupera el proveedor registrado en las bases de datos de prueba. También se verificó la recuperación de las relaciones principales. La validación integral todavía no está terminada debido a los errores de expansión anidada y a que faltan las pruebas específicas de filtros y paginación.
