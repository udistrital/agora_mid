## Endpoints

Las siguientes rutas distinguen la operación por método HTTP:

| Método | Ruta | Función |
| --- | --- | --- |
| `POST` | `/v1/registro/persona-natural` | Crear o reutilizar tercero y registrar proveedor natural. |
| `GET` | `/v1/registro/persona-natural?tercero_id={id}` | Consultar identidad, identificaciones, proveedor y datos naturales integrados. |
| `PUT` | `/v1/registro/persona-natural?tercero_id={id}` | Actualización parcial de campos permitidos para una persona natural. |
| `POST` | `/v1/registro/persona-juridica` | Crear o reutilizar tercero y registrar proveedor jurídico. |
| `GET` | `/v1/registro/persona-juridica?tercero_id={id}` | Consultar identidad, identificaciones, proveedor y datos jurídicos integrados. |
| `PUT` | `/v1/registro/persona-juridica?tercero_id={id}` | Actualización parcial de campos permitidos para una persona jurídica. |

La consulta integrada responde con `tercero_id`, `tipo_registro`, `terceros`, `datos_identificacion`, `proveedor` y el bloque `proveedor_natural` o `proveedor_juridico`. Los bloques procedentes de los CRUD conservan sus nombres JSON originales.

## Contrato de actualización parcial

El `PUT` recibe `tercero_id` en el cuerpo o en la consulta y uno o más bloques. Solo los campos presentes se actualizan; los valores omitidos se recuperan del recurso vigente y se conservan en el `PUT` dirigido al CRUD propietario.

Ejemplo de persona natural:

```json
{
  "tercero_id": 25,
  "terceros": { "primer_nombre": "Ana María" },
  "datos_identificacion": { "id": 99, "digito_verificacion": 9 },
  "proveedor": { "descripcion_portafolio": "Servicios profesionales" },
  "proveedor_natural": { "experiencia_laboral_meses": 24 }
}
```

Campos admitidos en `terceros`: `nombre_completo`, `primer_nombre`, `segundo_nombre`, `primer_apellido`, `segundo_apellido`, `fecha_nacimiento`, `lugar_origen`. Tipo y número de documento no se modifican por este endpoint. Para `datos_identificacion` se requiere el `id` que devuelve la consulta integrada. Se admiten `digito_verificacion`, `ciudad_expedicion` y `fecha_expedicion`. El tipo y número documental pueden cambiarse únicamente juntos (`codigo_tipo_documento` + `numero_documento`): el MID resuelve dinámicamente el tipo y verifica que el nuevo par no exista en otra identificación activa.

Campos admitidos en `proveedor`: `descripcion_portafolio`.

Campos admitidos en `proveedor_natural`: `perfil_declarado_id`, `experiencia_laboral_meses`, `experiencia_profesional_meses`.

Campos admitidos en `proveedor_juridico`: `nombre_comercial`, `matricula_mercantil`, `fecha_constitucion`, `fecha_renovacion`, `reporta_beneficios_finales`, `cotiza_bolsa`, `requiere_revisor_fiscal`.

Si se presenta un fallo después de que otro servicio ya actualizó datos, el MID informa que la actualización puede ser parcial; no intenta borrar una persona compartida ni ejecutar compensaciones destructivas.

## Alcance y pendientes

- La creación conserva el flujo previo de identidad en Terceros y los datos de proveedor en Agora.
- La consulta integrada y actualización parcial usan los endpoints del CRUD disponibles en el contrato de desarrollo revisado.
- La caracterización, los contactos generales, la seguridad social y algunos catálogos parametrizados requieren aún cerrar y probar el contrato específico de Terceros/Parámetros antes de incorporarlos.
- Las vinculaciones contractuales no pertenecen al alcance del MID para registro de personas.
