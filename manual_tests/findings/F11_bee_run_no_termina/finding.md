# F11: bee run no termina cuando el middleware sale

Estado: abierto. Operativo.
Solución: propuesta. Ver `## Propuesta`.
Gravedad: baja. Clasificación propuesta, ver utils/README.md.

## Explicación
Si el binario sale con `log.Fatalf`, `bee run` sigue corriendo como vigilante de archivos y el puerto no se libera.

## Procedimiento de trabajo
Para las pruebas, compilar y arrancar el binario directamente (ver `../../manual_tests.md`, sección 3). Antes de arrancar, revisar `pgrep -af bee` y matar los procesos que no correspondan.

## Propuesta

Arrancar el binario directamente, no con `bee run`. Ver `## Procedimiento de trabajo`.
