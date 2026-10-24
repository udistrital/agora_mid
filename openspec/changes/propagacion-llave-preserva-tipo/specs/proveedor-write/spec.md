# Spec Delta

## ADDED Requirements

### Requirement: Llaves numéricas en el subárbol materializado
El sistema SHALL devolver, en el `Data` materializado de `POST`/`PATCH
/v1/proveedor`, las llaves foráneas y las PK compartidas del subárbol con su tipo
numérico, coherente con el contrato de los CRUD dueños.

#### Scenario: FK devueltas como número
- **WHEN** una escritura materializa el subárbol y devuelve `Data` con la FK
  ascendente del proveedor (`TerceroId`) y las FK descendentes de las colecciones
  (`Contacto.ProveedorId`, `CuentaBancaria.ProveedorId`) o de
  `DatosIdentificacion.TerceroId`
- **THEN** cada una de esas llaves se serializa como número en el `Data`
  devuelto, no como cadena de texto

#### Scenario: PK compartida devuelta como número
- **WHEN** una escritura materializa una rama de detalle con PK compartida
  (`ProveedorNatural` o `ProveedorJuridico`) cuyo `Id` hereda del proveedor
- **THEN** el `Id` heredado se serializa como número en el `Data` devuelto
