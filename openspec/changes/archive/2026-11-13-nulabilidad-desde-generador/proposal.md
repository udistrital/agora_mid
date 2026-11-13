# Proposal

## Why

El generador de configuración no emite la nulabilidad de los campos, de modo que la
validación de escritura que rechaza `null` en un campo escalar no nulable (requisito
de `proveedor-write`) se apoya hoy en un marcador de posición: un script en Python
(`services/generar_campos_nulables.py`) que re-parsea el DBML por su cuenta, un mapa
Go generado a mano (`services/campos_nulables_gen.go`) y una validación
(`services/nulos.go`) que infiere la tabla dueña de cada campo por convención de
nombres. Son dos fuentes de verdad del mismo hecho del DBML y una resolución de tabla
frágil que el propio código marca con `TODO`. El DBML ya es la única entrada confiable
del generador; la nulabilidad debe salir de allí, por el mismo camino que los demás
roles de campo.

## What Changes

- El generador deriva la nulabilidad de cada campo del atributo `[not null]` del DBML
  y la emite en la tabla de campos tipada de cada objeto, junto a `readable`,
  `writable` e `immutable`. Un campo con `[not null]` (incluida la PK) se emite como no
  nulable; cualquier otro, como nulable.
- La nulabilidad es un hecho de base de datos, independiente de `required` (obligación
  del cliente). Este cambio **no** deriva ni aplica `required`; ese campo permanece sin
  poblar y fuera de alcance.
- El cargador de configuración acepta la nulabilidad por campo en la tabla de campos
  tipada y la proyecta al árbol efectivo compuesto, junto a los demás roles.
- El motor de escritura genérico rechaza con `400`, antes de cualquier escritura, un
  `null` enviado en un campo escalar no nulable, usando la nulabilidad proveniente del
  árbol efectivo compuesto en lugar de un mapa por convención de nombres.
- **BREAKING** (interno, sin cambio de contrato público): se eliminan el script
  `services/generar_campos_nulables.py`, el archivo generado
  `services/campos_nulables_gen.go` y la validación `validarNulos` con su resolución de
  tabla por convención de nombres en `services/nulos.go`.
- Se regenera la configuración bajo `config/` para incorporar la nulabilidad por campo;
  la salida regenerada se versiona como parte del cambio.

## Capabilities

### New Capabilities
<!-- Ninguna. -->

### Modified Capabilities
- `config-generation`: El generador deriva la nulabilidad de cada campo del atributo
  `[not null]` del DBML y la emite en la tabla de campos tipada, como una bandera de
  rol por campo más.
- `declarative-write-engine`: La nulabilidad por campo se suma a los roles que
  provienen de la misma tabla de campos tipada del árbol efectivo compuesto, y el motor
  la usa para rechazar `null` en un escalar no nulable antes de materializar, en lugar
  de una fuente paralela por convención de nombres.

## Impact

- **Generador** (`utils/generador_configuracion/`): el parser del DBML retiene el
  atributo `[not null]` de cada columna; la construcción de campos emite la nulabilidad;
  el README documenta la bandera. La salida determinista y la validación contra el DBML
  no cambian de forma.
- **Configuración generada** (`config/`): cada campo de cada objeto gana la bandera de
  nulabilidad; todos los archivos generados cambian y se versionan regenerados.
- **Cargador** (`config/tabla_campos.go`, `config/relaciones*.go`): la tabla de campos
  tipada acepta la nulabilidad por campo y la proyecta al `WriteConfig` del árbol
  efectivo.
- **Motor de escritura** (`helper/write_engine.go`): la validación previa a la escritura
  rechaza `null` en un escalar no nulable con la nulabilidad del árbol efectivo.
- **Código eliminado** (`services/`): `generar_campos_nulables.py`,
  `campos_nulables_gen.go` y el camino `validarNulos`/`camposNulables`/`snakeDeClave`,
  junto con su invocación en `services/proveedor_write.go`.
- **Contrato público**: sin cambios observables. El comportamiento `null` en no nulable
  -> `400` sin escrituras se conserva; solo cambia su fuente de verdad.
