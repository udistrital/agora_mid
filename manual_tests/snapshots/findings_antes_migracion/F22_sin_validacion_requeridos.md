# F22: no se validan los campos requeridos de un hijo

Estado: abierto. Reproducido en la corrida 4: P11 responde 200.

## Explicación
`POST /v1/proveedor` con un `Contacto` sin `Proposito` crea el contacto con `proposito = ''`. El contrato pide 400 o 502 para un campo requerido ausente.

## Causa
Ni el MID ni el CRUD validan campos requeridos de los hijos. La base acepta la cadena vacía.

## Propuesta
Validar en el MID los campos `required` del objeto antes de escribir, y devolver 400 con el nombre del campo.
