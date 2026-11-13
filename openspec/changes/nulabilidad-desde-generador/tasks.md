# Tasks

## 1. Parser del DBML retiene la no-nulabilidad

- [x] 1.1 Escribir la prueba del parser (con marcador de rojo) que verifica que una columna con `[not null]` se parsea como no nulable, una columna sin atributos como nulable, la llave primaria como no nulable, y que el nombre y el tipo de una columna con otros atributos no sufren regresión, y confirmar que falla
- [x] 1.2 Capturar el contenido del corchete de atributos en `reColumna` y añadir la no-nulabilidad a `Columna` (verdadera ante `not null` o llave primaria), y verificar que la prueba 1.1 pasa y se retira su marcador

## 2. El generador emite la nulabilidad por campo

- [x] 2.1 Escribir la prueba de construcción (con marcador de rojo) que verifica que `construirCampos` emite `nullable` por campo como el inverso de la no-nulabilidad de la columna, que un campo con `[not null]` y la llave primaria se emiten no nulables, y que la emisión es idéntica byte a byte entre dos ejecuciones en la posición fija del campo, y confirmar que falla
- [x] 2.2 Añadir `Nullable` a `CampoConfig`, emitirlo en `construirCampos` como `!NoNula`, y verificar que la prueba 2.1 pasa y se retira su marcador
- [x] 2.3 Documentar la bandera `nullable` en el README del generador (origen en `[not null]`, independiente de `required`), y verificar que el README coincide con el comportamiento de `Generar`

## 3. El cargador acepta y proyecta la nulabilidad

- [x] 3.1 Escribir la prueba de la tabla de campos (con marcador de rojo) que verifica que `CampoObjeto` decodifica `nullable`, que `CamposNoNulables()` devuelve los campos no nulables en orden de declaración, y que un contenido sin la clave toma el valor por omisión esperado, y confirmar que falla
- [x] 3.2 Añadir `Nullable` a `CampoObjeto` y el proyector `CamposNoNulables()` a `TablaCampos`, y verificar que la prueba 3.1 pasa y se retira su marcador
- [x] 3.3 Escribir la prueba de carga (con marcador de rojo) que verifica que `WriteConfig` recibe los campos no nulables tanto en la raíz (`aplicarRaiz`) como en un objeto anidado no raíz de la composición por relación, y confirmar que falla
- [x] 3.4 Añadir `NonNullableFields` a `WriteConfig` y poblarlo desde `CamposNoNulables()` en `aplicarRaiz` y en la composición por relación, y verificar que la prueba 3.3 pasa y se retira su marcador

## 4. Configuración regenerada con nulabilidad

- [ ] 4.1 Regenerar `config/` con el generador y verificar que cada campo de cada objeto generado lleva `nullable`, que dos ejecuciones producen archivos idénticos byte a byte, y versionar la salida regenerada

## 5. El motor rechaza null en escalar no nulable

- [ ] 5.1 Escribir la prueba del motor (con marcador de rojo) que verifica que `validarNivel` devuelve `*ErrorEscritura` ante un `null` en un campo escalar escribible no nulable, que no lo rechaza en un campo nulable, que un `null` en una FK derivada o en una relación no se rechaza, y que el rechazo ocurre antes de cualquier lectura o escritura, y confirmar que falla
- [ ] 5.2 Añadir en `validarNivel` el rechazo de `null` en un campo de `NonNullableFields` que no sea relación, FK derivada reconocida ni `Id`, y verificar que la prueba 5.1 pasa y se retira su marcador

## 6. Retiro del marcador y verificación de integración

- [ ] 6.1 Eliminar `services/generar_campos_nulables.py`, `services/campos_nulables_gen.go` y el camino `validarNulos`/`camposNulables`/`snakeDeClave` de `services/nulos.go`, retirar su invocación en `services/proveedor_write.go`, y verificar con `grep` que no quedan referencias a esos símbolos
- [ ] 6.2 Verificar que `test/nulos_test.go` pasa sin cambios de comportamiento observable: `null` en no nulable responde `400` sin escrituras; `null` en nulable se fija; `null` en FK relacional se trata como ausente; la colección vacía no elimina
- [ ] 6.3 Ejecutar `go test ./...` y `go build ./...` y verificar que toda la suite pasa sin marcadores de rojo remanentes
