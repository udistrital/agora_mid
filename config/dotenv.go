package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

// CargarDotEnv carga variables desde un archivo .env sin sobrescribir las que ya
// existen en el entorno del proceso. Devuelve cuántas variables nuevas definió.
//
// El formato aceptado es una variable por línea:
//
//	CLAVE=valor
//	export CLAVE="valor"
//
// Se ignoran las líneas vacías y las que inician con #. Los valores entre
// comillas simples o dobles se guardan sin las comillas. Si el archivo no
// existe, no es un error: en producción se espera que las variables vengan del
// entorno del proceso.
func CargarDotEnv(ruta string) (int, error) {
	archivo, err := os.Open(ruta)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("abriendo %s: %w", ruta, err)
	}
	defer archivo.Close()

	definidas := 0
	var primerError error
	scanner := bufio.NewScanner(archivo)
	numeroLinea := 0

	for scanner.Scan() {
		numeroLinea++
		clave, valor, ok, errLinea := interpretarLineaDotEnv(scanner.Text())
		if errLinea != nil {
			if primerError == nil {
				primerError = fmt.Errorf("%s:%d: %w", ruta, numeroLinea, errLinea)
			}
			continue
		}
		if !ok {
			continue
		}
		if _, existe := os.LookupEnv(clave); existe {
			continue
		}
		if err := os.Setenv(clave, valor); err != nil {
			if primerError == nil {
				primerError = fmt.Errorf("%s:%d: definiendo %s: %w", ruta, numeroLinea, clave, err)
			}
			continue
		}
		definidas++
	}

	if err := scanner.Err(); err != nil {
		return definidas, errors.Join(primerError, fmt.Errorf("leyendo %s: %w", ruta, err))
	}
	return definidas, primerError
}

// interpretarLineaDotEnv convierte una línea del archivo en clave y valor. La
// cuarta respuesta indica si la línea contiene una variable.
func interpretarLineaDotEnv(linea string) (clave, valor string, ok bool, err error) {
	linea = strings.TrimSpace(strings.TrimSuffix(linea, "\r"))
	linea = strings.TrimPrefix(linea, "\ufeff")
	if linea == "" || strings.HasPrefix(linea, "#") {
		return "", "", false, nil
	}

	linea = strings.TrimSpace(strings.TrimPrefix(linea, "export "))
	clave, valor, encontrado := strings.Cut(linea, "=")
	if !encontrado {
		return "", "", false, fmt.Errorf("línea sin '='")
	}

	clave = strings.TrimSpace(clave)
	if clave == "" {
		return "", "", false, fmt.Errorf("variable sin nombre")
	}
	if strings.ContainsAny(clave, " \t") {
		return "", "", false, fmt.Errorf("nombre de variable inválido: %q", clave)
	}

	valor = strings.TrimSpace(valor)
	valor = quitarComillas(valor)
	return clave, valor, true, nil
}

// quitarComillas elimina un par de comillas simples o dobles que envuelva todo
// el valor. No interpreta secuencias de escape: el contenido se conserva tal
// cual, salvo las comillas externas.
func quitarComillas(valor string) string {
	if len(valor) < 2 {
		return valor
	}
	primera := valor[0]
	ultima := valor[len(valor)-1]
	if primera == ultima && (primera == '\'' || primera == '"') {
		return valor[1 : len(valor)-1]
	}
	return valor
}
