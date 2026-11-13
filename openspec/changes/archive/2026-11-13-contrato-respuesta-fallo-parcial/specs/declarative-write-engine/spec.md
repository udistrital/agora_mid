# Spec Delta

## MODIFIED Requirements

### Requirement: Reporte tolerante de errores de escritura
El sistema SHALL honrar la política de error de relación configurada durante la
escritura: en modo tolerante conserva lo ya materializado y reporta avisos; en
modo estricto detiene la cascada, y el código y el cuerpo de la respuesta dependen
de si ya se materializó alguna entidad.

#### Scenario: Fallo de relación tolerado durante la escritura
- **WHEN** una relación no bloqueante falla y la política es tolerante
- **THEN** el motor conserva las entidades ya materializadas, continúa con las
  demás y agrega un aviso no sensible con la ruta lógica de la relación

#### Scenario: Fallo de relación bloqueante
- **WHEN** una relación bloqueante falla o la política es estricta, y aún no se ha
  materializado ninguna entidad
- **THEN** el motor detiene la cascada y reporta el error sin exponer URLs,
  credenciales ni cuerpos remotos, sin devolver ningún subárbol parcial

#### Scenario: Fallo estricto después de materializar (stop-abort)
- **WHEN** una relación bloqueante falla o la política es estricta después de que
  el motor materializó al menos una entidad
- **THEN** el motor detiene la cascada, no intenta los nodos posteriores y devuelve
  el subárbol ya materializado con un aviso tipado que nombra el nodo de parada,
  sin exponer URLs, credenciales ni cuerpos remotos
