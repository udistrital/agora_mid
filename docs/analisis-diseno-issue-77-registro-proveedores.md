# Análisis y diseño técnico - Issue #77
## Implementación de endpoints de registro de proveedores en MS MID

**Proyecto:** agora_mid
**Issue:** #77 - Implementación endpoints de registro de proveedores en MS MID
**Rama:** feature/issue-77-endpoints-registro-proveedores
**Fecha de análisis:** 18 de noviembre de 2026
**Estado:** Análisis técnico y diseño preliminar

---

## 1. Objetivo

Analizar la arquitectura existente de escritura de proveedores en MS MID y establecer una propuesta técnica para implementar endpoints especializados que permitan integrar los datos del proveedor agrupados por fuente de datos:

- Datos básicos.
- Caracterización.
- Datos complementarios.

El análisis busca identificar los componentes reutilizables, las relaciones entre entidades, las diferencias entre los contratos configurados y los modelos reales de los servicios CRUD, así como las pruebas necesarias para validar la implementación.

Este documento registra el análisis y las propuestas de diseño. No representa funcionalidades nuevas ya implementadas.

## 2. Alcance de la issue

La descripción de la issue plantea:

> Se requiere realizar el análisis, diseño y estructuración de endpoints especializados para la integración de datos del proveedor agrupando los datos por fuente de datos y endpoints a consumir (datos básicos, caracterización, datos complementarios).

Subtareas:

1. Implementación de endpoints de registro de datos básicos (terceros y proveedores).
2. Implementación de endpoints de caracterización.
3. Implementación de pruebas unitarias.

Criterio de aceptación:

- Implementación de endpoints de registro de datos del proveedor.

Definición de terminado:

- Desarrollo local.
- Push a rama Feature.
- Pruebas funcionales locales.
- PR hacia Develop.
- Criterios de aceptación cumplidos.
- Documentación de la issue.
- Aprobación del Scrum Master o líder técnico.

## 3. Arquitectura actual de escritura en MS MID

### 3.1. Enrutamiento

En `main.go` se configura el handler completo de proveedores mediante:

- `ProveedorService`.
- `ProveedorWriteService`.
- `NewProveedorHandlerCompleto`.

Las rutas actuales de escritura son:

| Método | Ruta | Comportamiento |
|---|---|---|
| POST | `/v1/proveedor` | Escritura de proveedor |
| PATCH | `/v1/proveedor` | Actualización de proveedor |

Ambas rutas pasan por `enrutarProveedor()` y delegan en `h.Escribir()`.

Las consultas se encuentran separadas:

| Método | Ruta |
|---|---|
| POST | `/v1/proveedor/consulta` |
| POST | `/v1/proveedor/consulta/lista` |

### 3.2. Servicio de escritura

Archivo: `services/proveedor_write.go`

El servicio `ProveedorWriteService`:

- Valida que el cuerpo no sea nulo.
- Valida que `TipoRegistroId` sea positivo.
- Comprueba la coherencia entre el tipo de registro y la rama natural o jurídica.
- Elimina la rama incompatible.
- Delega la escritura al motor genérico `helper.Write`.

El servicio utiliza una configuración declarativa y un registro de resolvers.

### 3.3. Motor genérico de escritura

Archivo: `helper/write_engine.go`

El motor admite tres mecanismos de propagación de identificadores:

| Mecanismo | Descripción |
|---|---|
| `fk` | Propaga el identificador del padre al campo de clave foránea del hijo. |
| `shared_pk` | Propaga el identificador del padre como clave primaria del hijo. |
| `fk_inbound` | Materializa primero la entidad referenciada y propaga su identificador hacia el padre. |

El motor realiza una fase previa de resolución de relaciones entrantes y posteriormente materializa las entidades.

La implementación permite reutilizar lógica de escritura para entidades relacionadas.

**Limitación identificada:** el motor no proporciona una transacción distribuida ni garantiza rollback de operaciones que ya fueron ejecutadas cuando falla una operación posterior.

### 3.4. Resolución de entidades

Archivo: `services/write_resolver.go`

`EntityWriteResolver.Materialize` permite:

- Resolver una entidad existente mediante su identificador.
- Buscar una entidad mediante una clave natural cuando está configurada.
- Ejecutar PATCH si encuentra un registro existente.
- Ejecutar POST cuando no existe.
- Recuperar el identificador de la respuesta para propagarlo a las relaciones.

El resolver también dispone de `Recuperar`, utilizado para consultar entidades existentes antes de materializarlas.

## 4. Inventario de resolvers existentes

Archivo: `services/write_resolvers_nucleo.go`

Se identificaron los siguientes resolvers configurados:

| Entidad | Servicio CRUD | Endpoint |
|---|---|---|
| Proveedor | gestion_personas_crud | `/v1/proveedor` |
| Tercero | terceros | `/v1/tercero` |
| DatosIdentificacion | terceros | `/v1/datos_identificacion` |
| ProveedorNatural | gestion_personas_crud | `/v1/proveedor_natural` |
| ProveedorJuridico | gestion_personas_crud | `/v1/proveedor_juridico` |
| Contacto | gestion_personas_crud | `/v1/contacto` |
| CuentaBancaria | gestion_personas_crud | `/v1/cuenta_bancaria` |

No se identificaron resolvers específicos en este registro para:

- ActividadEconomicaProveedor.
- Representacion.
- PerfilProveedor.
- TituloProveedorNatural.
- PersonaJuridicaTipoEntidad.
- Documento.
- EstadoProveedor.

La existencia de rutas CRUD para estas entidades no implica que ya estén integradas al motor de escritura del MID.

## 5. Inventario de entidades disponibles en Gestión de Personas CRUD

Se inspeccionaron los modelos y las rutas del repositorio `gestion_personas_crud`.

Las rutas identificadas incluyen:

- `/v1/proveedor`
- `/v1/proveedor_natural`
- `/v1/proveedor_juridico`
- `/v1/actividad_economica_proveedor`
- `/v1/representacion`
- `/v1/perfil_proveedor`
- `/v1/titulo_proveedor_natural`
- `/v1/persona_juridica_tipo_entidad`
- `/v1/documento`
- `/v1/estado_proveedor`
- `/v1/contacto`
- `/v1/cuenta_bancaria`

Se confirma que existen controladores CRUD para las entidades revisadas. Sin embargo, su disponibilidad no significa que todas estén actualmente integradas en MS MID.

## 6. Relaciones e integridad referencial

Archivo inspeccionado: `gestion_personas_crud/database/agora_proveedores.sql`.

Relaciones identificadas:

| Tabla dependiente | Campo | Referencia |
|---|---|---|
| proveedor_natural | id | proveedor.id |
| proveedor_juridico | id | proveedor.id |
| contacto | proveedor_id | proveedor.id |
| cuenta_bancaria | proveedor_id | proveedor.id |
| cuenta_bancaria | contacto_id | contacto.id |
| actividad_economica_proveedor | proveedor_id | proveedor.id |
| representacion | proveedor_id | proveedor.id |
| perfil_proveedor | proveedor_id | proveedor.id |
| titulo_proveedor_natural | proveedor_id | proveedor.id |
| persona_juridica_tipo_entidad | persona_juridica_id | proveedor_juridico.id |
| documento | proveedor_id | proveedor.id |
| estado_proveedor | proveedor_id | proveedor.id |

Las tablas `proveedor_natural` y `proveedor_juridico` utilizan una clave primaria compartida con `proveedor`.

La configuración actual emplea `shared_pk` para las especializaciones natural y jurídica.

Las relaciones de `cuenta_bancaria` requieren especial atención porque dependen tanto del proveedor como del contacto.

## 7. Revisión de configuración YAML

Archivo: `config/relaciones_proveedor.yaml`

La configuración actual declara el servicio `proveedor`, sus campos de escritura, las relaciones y sus reglas de propagación.

Se identificaron relaciones para:

- Tercero.
- DatosIdentificacion.
- TipoContribuyente.
- ProveedorNatural.
- ProveedorJuridico.
- Contacto.
- CuentaBancaria.

La configuración contempla relaciones entrantes para tercero y datos de identificación, además de claves compartidas para las especializaciones natural y jurídica.

### 7.1. Diferencias entre YAML y modelos Go

Se encontraron las siguientes incompatibilidades:

| Entidad | Configuración YAML | Modelo Go |
|---|---|---|
| ProveedorNatural | `ExperienciaLaboralMeses`, `ExperienciaProfesionalMeses` | `ExperienciaLaboral`, `ExperienciaProfesional` |
| ProveedorJuridico | `NombreComercial`, `MatriculaMercantil`, `FechaConstitucion`, entre otros | `ProcedenciaEmpresaId`, `CiudadOrigenId`, `PaisOrigenId`, `CodigoPostal`, `TipoConformacionId`, entre otros |
| Contacto | `Finalidad`, `TipoCanal`, `EsPrincipal` | `Proposito`, `TipoCanalContactoId` |
| CuentaBancaria | `NombreTitular`, `CiudadAperturaId`, `EsPrincipal` | `ContactoId`, `TipoCuentaBancariaId`, `MontoCapitalAutorizado`, entre otros |

Estas diferencias deben resolverse antes de reutilizar la configuración actual como contrato definitivo de escritura.

### 7.2. Diferencia en la ruta de perfil

La configuración YAML inspeccionada contiene una referencia a `/v1/perfil`.

El router de `gestion_personas_crud` expone `/v1/perfil_proveedor` para la entidad `PerfilProveedor`.

Se debe distinguir la entidad de catálogo Perfil de la relación PerfilProveedor y verificar a qué servicio pertenece cada ruta antes de modificar la configuración.

## 8. Agrupación funcional preliminar

La siguiente clasificación es una propuesta de trabajo basada en las entidades y relaciones identificadas. No constituye todavía una definición funcional aprobada.

### 8.1. Datos básicos

Entidades propuestas:

- Tercero.
- DatosIdentificacion.
- Proveedor.
- ProveedorNatural.
- ProveedorJuridico.

Esta agrupación permite registrar la información de identificación del tercero y la información principal del proveedor, incluyendo la especialización según su tipo de registro.

### 8.2. Caracterización

Entidades propuestas:

- ActividadEconomicaProveedor.
- Representacion.
- PerfilProveedor.
- TituloProveedorNatural.
- PersonaJuridicaTipoEntidad.

La clasificación debe validarse con el líder técnico o con la especificación funcional del módulo.

### 8.3. Datos complementarios

Entidades propuestas:

- Contacto.
- CuentaBancaria.
- Documento.

`EstadoProveedor` queda pendiente de clasificación funcional, pues corresponde a información de estado o ciclo de vida del registro.

## 9. Contrato de entrada y relaciones JSON

Los controladores CRUD inspeccionados deserializan el cuerpo JSON directamente en los modelos Go.

Algunos modelos representan relaciones mediante punteros a estructuras, por ejemplo:

- `ActividadEconomicaProveedor.ProveedorId *Proveedor`.
- `Contacto.ProveedorId *Proveedor`.
- `CuentaBancaria.ProveedorId *Proveedor`.
- `CuentaBancaria.ContactoId *Contacto`.
- `PersonaJuridicaTipoEntidad.PersonaJuridicaId *ProveedorJuridico`.

Por tanto, no se debe asumir que todos los campos de relación reciben un identificador numérico plano. La representación JSON debe verificarse contra los modelos y las pruebas reales del CRUD.

También se identificó que `CuentaBancaria.TipoCuentaBancariaId` está declarado como booleano en el modelo Go, pese al sufijo `Id`. Se debe contrastar con el esquema SQL y la definición funcional antes de proponer cualquier cambio.

## 10. Pruebas existentes y cobertura identificada

Archivo: `services/proveedor_write_test.go`

Se inspeccionó una prueba de creación completa que utiliza un servidor HTTP simulado para validar la orquestación de:

- Proveedor.
- Tercero.
- DatosIdentificacion.
- ProveedorNatural.
- Contacto.

La prueba comprueba propagación de identificadores, orden de materialización y exclusión de la rama incompatible.

También se identificaron pruebas de coherencia entre el tipo de registro y la rama natural o jurídica.

**Limitación de la evidencia:** las pruebas utilizan una configuración construida específicamente para el test. Algunos campos usados en ella no coinciden con los modelos reales del CRUD. Por lo tanto, demuestran el comportamiento del motor de escritura bajo esa configuración, pero no acreditan por sí solas la compatibilidad de todos los contratos reales.

Queda pendiente completar el inventario de funciones de prueba y ejecutar la suite correspondiente.

## 11. Propuesta técnica preliminar

Se propone reutilizar el motor genérico de escritura y los resolvers existentes siempre que sus contratos sean compatibles con los modelos CRUD.

Antes de implementar:

1. Definir el contrato de entrada de cada grupo.
2. Confirmar si los endpoints serán independientes o si habrá un endpoint orquestador.
3. Validar las entidades que corresponden a caracterización y datos complementarios.
4. Ajustar las diferencias entre YAML y modelos Go.
5. Incorporar los resolvers que hagan falta para las entidades no integradas.
6. Definir el comportamiento de creación y actualización.
7. Diseñar pruebas para relaciones, claves compartidas y errores parciales.

No se propone duplicar el motor genérico de escritura si la arquitectura existente permite extenderlo.

La definición final de rutas y contratos queda pendiente de validación funcional y técnica.

## 12. Matriz de actividades pendientes

| Actividad | Estado |
|---|---|
| Inspección de arquitectura MID | Realizada |
| Inventario de resolvers existentes | Realizado |
| Inspección de rutas CRUD | Realizada |
| Revisión de modelos Go | Realizada |
| Revisión de relaciones SQL | Realizada |
| Identificación de diferencias YAML/modelos | Realizada |
| Clasificación funcional definitiva | Pendiente |
| Definición de rutas especializadas | Pendiente |
| Definición de contratos JSON | Pendiente |
| Ajuste de configuración YAML | Pendiente |
| Implementación de resolvers faltantes | Pendiente |
| Implementación de endpoints | Pendiente |
| Pruebas unitarias | Pendiente |
| Pruebas funcionales locales | Pendiente |
| Documentación final y PR | Pendiente |

## 13. Riesgos técnicos

1. Incompatibilidad entre campos configurados en YAML y atributos reales de los modelos CRUD.
2. Errores en la propagación de identificadores para relaciones con clave foránea o clave primaria compartida.
3. Dependencias entre entidades, especialmente cuenta bancaria y contacto.
4. Ausencia de transacción distribuida y posibilidad de escrituras parciales.
5. Contratos JSON no verificados para relaciones representadas mediante punteros a estructuras.
6. Clasificación funcional de entidades todavía pendiente de aprobación.

## 14. Conclusión

MS MID dispone de una arquitectura genérica de escritura que permite reutilizar lógica de materialización y propagación de identificadores para entidades relacionadas.

Existe un flujo de escritura para proveedor que integra tercero, datos de identificación, especialización natural o jurídica, contacto y cuenta bancaria.

Sin embargo, la revisión identificó diferencias entre la configuración YAML y los modelos reales del CRUD, además de entidades que cuentan con rutas CRUD pero todavía no tienen resolvers específicos integrados en el flujo actual.

El siguiente paso consiste en validar la agrupación funcional y los contratos de los endpoints especializados. Con esas definiciones se podrá completar el diseño, implementar los resolvers y endpoints faltantes y desarrollar las pruebas unitarias y funcionales exigidas por la issue.
