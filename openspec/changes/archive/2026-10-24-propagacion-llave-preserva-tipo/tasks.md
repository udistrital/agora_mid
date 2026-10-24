# Tasks

## 1. Preservar el tipo de la llave en el motor de escritura

- [x] 1.1 En `helper/write_engine.go`, cambiar `propagarLlave` para recibir
  `padreID any` y asignar `hijo[rel.RelatedField]`/`hijo["Id"]` con ese valor sin
  convertir; sustituir la guarda `padreID == ""` por `padreID == nil`. Verificar
  con `go build ./...`.
- [x] 1.2 En `materializarHijos`, declarar `var padreID any` (nil) y asignarlo con
  `padre["Id"]` tal cual (sin `aCadena`) solo cuando exista y no sea nil; ajustar
  la firma de `materializarValorRelacion` a `padreID any`. Verificar con
  `go build ./...`.
- [x] 1.3 En `materializarInbound`, propagar `nodo[rel.LocalField] = referida["Id"]`
  (entero del resolver) en lugar de `aCadena(id)`. Verificar con `go build ./...`.
- [x] 1.4 Confirmar que `aCadena` ya no se usa en la propagación y permanece solo
  en `filtrarPorRelatedField` (lectura) y en `valoresClaveNatural` (query de clave
  natural). Verificar con `grep -n aCadena helper/write_engine.go helper/expand.go`
  y revisar que los usos restantes son los esperados.
- [x] 1.5 Endurecer `helper/write_engine_test.go` para afirmar que las llaves
  propagadas (`PadreId` por FK e `Id` por PK compartida) son de tipo numérico, no
  string. Verificar con `go test ./helper/...`.

## 2. Verificar el tipo numérico extremo a extremo

- [x] 2.1 En `test/integration_test.go`, reemplazar las comparaciones de FK con
  `comoTexto()` por aserciones de tipo numérico (`float64` tras decodificar) para
  `TerceroId` y `Contacto[].ProveedorId`, y afirmar el `Id` numérico heredado en
  `ProveedorNatural`. Verificar con `go test ./test/...`.
- [x] 2.2 En `test/e2e_test.go`, afirmar que `proveedor.TerceroId` es numérico e
  igual a `tercero.Id`, sin normalizar con `comoTexto()`. Verificar con
  `go test ./test/...`.
- [x] 2.3 Revisar si `comoTexto()` queda sin uso en el paquete `test`; si es así,
  eliminarlo o dejarlo solo donde siga siendo necesario. Verificar con
  `go vet ./test/...` y `go test ./test/...`.

## 3. Documentación del contrato público (swagger) fiel al comportamiento

- [x] 3.1 En `swagger/swagger.yml`, redefinir `ProveedorMaterializado` como el
  subárbol materializado real: entidades anidadas (`Tercero`,
  `DatosIdentificacion`, rama de detalle, `Contacto`, `CuentaBancaria`) con sus
  `Id`, las FK (`TerceroId`, `ProveedorId`) como `type: integer` y el campo
  `AutorModificacionId`. Verificar que el YAML es válido (p. ej.
  `python3 -c "import yaml,sys; yaml.safe_load(open('swagger/swagger.yml'))"`).
- [x] 3.2 Ajustar en `swagger/swagger.yml` cualquier otra referencia a las FK de
  escritura para que su tipo sea entero, coherente con el comportamiento. Verificar
  revisando que no queden FK de escritura documentadas como `type: string`.
- [x] 3.3 Actualizar `API.md` si describe la forma del `Data` de escritura, para
  que coincida con el subárbol materializado y los tipos numéricos. Verificar por
  lectura que `API.md` y el YAML describen el mismo `Data`.
- [x] 3.4 Confirmar que el JSON derivado del contrato queda coherente con el YAML
  (la regeneración ocurre al arranque del API según `openapi-publication`).
  Verificar con `go build ./...` y, si aplica, ejecutando la generación
  (`go test ./... -run Swagger` o el arranque) para comprobar que no hay divergencia.

## 4. Documentación de la suite de pruebas

- [x] 4.1 Actualizar `test/tests.md` con las salidas esperadas de tipo numérico de
  las FK/PK en los escenarios de creación y e2e. Verificar por lectura que las
  salidas descritas coinciden con las aserciones de los tests del grupo 2.

## 5. Verificación integral

- [x] 5.1 Ejecutar `go build ./...` y `go test ./...` y confirmar que todo el
  proyecto compila y la suite completa pasa en verde.
- [x] 5.2 Validar el cambio con `openspec validate "propagacion-llave-preserva-tipo"`
  y confirmar que es válido.
