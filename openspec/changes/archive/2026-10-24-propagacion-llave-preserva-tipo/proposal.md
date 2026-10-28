# Proposal

## Why

Al revisar la fidelidad de `swagger/swagger.yml` contra el comportamiento real
(capturando los payloads reales en las pruebas de integración) se descubrió que
el motor de escritura coacciona a **string** el `Id` que propaga en cascada: las
llaves foráneas (`proveedor.TerceroId`, `Contacto.ProveedorId`,
`DatosIdentificacion.TerceroId`) y las PK compartidas salen como `"201"` hacia los
CRUD dueños y hacia el cliente, cuando el contrato de esos CRUD y la propia
especificación de lectura (que exige "conservar la llave local original") esperan
un entero. La causa raíz es la reutilización del helper de comparación `aCadena`
—nacido en el camino de lectura solo para comparar llaves— dentro de la
propagación de escritura, donde el valor ya no se compara sino que se persiste y
se devuelve. Como nos comunicamos por JSON y el valor numérico es el mismo, la
corrección correcta es dejar de coaccionar el tipo en la propagación.

## What Changes

- Se elimina el uso de `aCadena` en los tres puntos de propagación de llave del
  motor de escritura (`helper/write_engine.go`): `materializarHijos`,
  `materializarInbound` y `propagarLlave`. El `Id` del padre se propaga con su
  tipo nativo (como `any`), no como texto. La guarda centinela `""` pasa a `nil`.
- `aCadena` permanece intacto para su propósito legítimo: comparación de llaves
  en lectura (`helper/expand.go`) y construcción de la query de clave natural
  (`valoresClaveNatural`), que es texto por naturaleza.
- Se endurecen las pruebas (`helper/write_engine_test.go`, `test/integration_test.go`,
  `test/e2e_test.go`) para **afirmar el tipo numérico** de las FK/PK propagadas,
  en lugar de normalizar con `comoTexto()` (que hoy enmascara el bug).
- Se actualiza `swagger/swagger.yml` (y su JSON derivado) para reflejar fielmente
  el comportamiento real de escritura: las FK (`TerceroId`, `ProveedorId`) como
  enteros, y `Data` como el **subárbol materializado completo** (no solo tres
  identificadores), incluyendo el campo de auditoría `AutorModificacionId` que el
  motor inyecta y devuelve.
- Se actualiza la documentación de la suite (`test/tests.md`) con las salidas
  esperadas de tipo numérico.

## Capabilities

### New Capabilities
<!-- Ninguna capacidad nueva. -->

### Modified Capabilities

- `declarative-write-engine`: se precisa el requisito de propagación de llave para
  exigir que la llave del padre se propague **preservando su tipo** (numérico),
  sin coaccionarla a texto.
- `proveedor-write`: se precisa que el `Data` materializado devuelto por
  `POST`/`PATCH` contiene el subárbol con las llaves (FK y PK compartida) en su
  tipo numérico.
- `openapi-publication`: se precisa que el contrato publicado describe el cuerpo
  materializado de escritura de forma fiel al comportamiento implementado (tipos
  de las FK y forma real de `Data`).

## Impact

- **Código**: `helper/write_engine.go` (eliminación de `aCadena` en propagación;
  `padreID` pasa de `string` a `any`; firma de `propagarLlave`; guarda `nil`).
  `services/write_resolver.go` ya normaliza el `Id` a entero en la frontera API
  (fuente del tipo canónico); `idEntero` se conserva tolerante a string como red
  de seguridad defensiva.
- **Contrato público**: `swagger/swagger.yml` y `swagger/swagger.json` derivado;
  coherencia con `API.md`.
- **Pruebas**: `helper/write_engine_test.go`, `test/integration_test.go`,
  `test/e2e_test.go` (aserciones de tipo) y `test/tests.md` (salidas esperadas).
- **Compatibilidad**: los consumidores que hoy reciben las FK como string verán
  un número. El valor JSON es equivalente (`"201"` -> `201`), pero un cliente que
  dependa estrictamente del tipo string debe ajustarse. No cambia el envoltorio
  `Success/Status/Message/Data` ni los códigos de respuesta.
