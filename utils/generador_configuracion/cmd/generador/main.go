// Comando generador produce a mano la configuración de relaciones a partir del
// DBML y de los sidecars de intención. No forma parte del arranque del API.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/udistrital/agora_mid/utils/generador_configuracion/internal/generador"
)

// main lee las entradas de -carpeta y escribe los objetos generados en -salida.
func main() {
	carpeta := flag.String("carpeta", "utils/generador_configuracion/entradas", "carpeta con el DBML y los sidecars.")
	dbml := flag.String("dbml", "agora_modelo_extendido.dbml", "nombre del DBML dentro de -carpeta.")
	salida := flag.String("salida", "", "carpeta donde se escriben los objetos generados (obligatoria).")
	flag.Parse()

	if *salida == "" {
		fmt.Fprintln(os.Stderr, "generador: falta -salida")
		os.Exit(2)
	}
	if err := generador.Generar(filepath.Join(*carpeta, *dbml), *carpeta, *salida); err != nil {
		fmt.Fprintf(os.Stderr, "generador: %v\n", err)
		os.Exit(1)
	}
}
