package generador

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// encabezadoGenerado se antepone a cada archivo para advertir que no se edita a mano.
// Los archivos que lo llevan son los que el generador puede eliminar al limpiar.
const encabezadoGenerado = "# Generado por utils/generador_configuracion. No editar a mano; regenerar desde el DBML y los sidecars.\n"

// EscribirObjetos escribe cada objeto en salida/<related_api>/objetos_<tabla>.yaml,
// creando las carpetas necesarias. La serialización sigue el orden de los campos de
// ObjetoConfig, así que la misma entrada produce los mismos bytes.
func EscribirObjetos(objetos []ObjetoConfig, salida string) error {
	for _, o := range objetos {
		if err := escribirYAML(filepath.Join(salida, o.RelatedAPI), "objetos_"+o.RelatedTable+".yaml", o); err != nil {
			return err
		}
	}
	return nil
}

// escribirYAML serializa datos en dir/nombre con el encabezado generado, creando dir
// si no existe.
func escribirYAML(dir, nombre string, datos any) error {
	contenido, err := yaml.Marshal(datos)
	if err != nil {
		return fmt.Errorf("serializando %s: %w", nombre, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creando %s: %w", dir, err)
	}
	ruta := filepath.Join(dir, nombre)
	if err := os.WriteFile(ruta, append([]byte(encabezadoGenerado), contenido...), 0o644); err != nil {
		return fmt.Errorf("escribiendo %s: %w", ruta, err)
	}
	return nil
}

// limpiarSalida elimina de salida los archivos YAML que llevan el encabezado generado
// y después las carpetas que quedan vacías. Los archivos sin ese encabezado se
// conservan. Si salida no existe, no hace nada.
func limpiarSalida(salida string) error {
	if _, err := os.Stat(salida); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	var generados, carpetas []string
	err := filepath.WalkDir(salida, func(ruta string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if ruta != salida {
				carpetas = append(carpetas, ruta)
			}
			return nil
		}
		if !strings.HasSuffix(ruta, ".yaml") {
			return nil
		}
		contenido, err := os.ReadFile(ruta)
		if err != nil {
			return err
		}
		if strings.HasPrefix(string(contenido), encabezadoGenerado) {
			generados = append(generados, ruta)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("leyendo la salida %s: %w", salida, err)
	}
	for _, ruta := range generados {
		if err := os.Remove(ruta); err != nil {
			return fmt.Errorf("eliminando %s: %w", ruta, err)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(carpetas)))
	for _, carpeta := range carpetas {
		// Solo se eliminan carpetas vacías; si conserva archivos ajenos, os.Remove falla.
		_ = os.Remove(carpeta)
	}
	return nil
}
