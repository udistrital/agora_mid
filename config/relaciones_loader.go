package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ServiceURLs mapea cada nombre lógico de servicio con su URL base, resuelta
// desde conf/app.conf. Se usa para validar que related_service exista.
type ServiceURLs map[string]string

// URLs construye el mapa de servicios lógicos a URL base a partir de la Config
// cargada desde conf/app.conf.
func (c Config) URLs() ServiceURLs {
	return ServiceURLs{
		"terceros":              c.TercerosCRUDURL,
		"parametros":            c.ParametrosCRUDURL,
		"gestion_personas_crud": c.GestionPersonasCRUDURL,
	}
}

// LoadRelaciones carga y valida todos los archivos config/relaciones_*.yaml del
// directorio indicado, resolviendo related_service contra urls. Un archivo
// ausente, inválido o con una relación inconsistente devuelve un error que
// identifica el archivo y, cuando aplica, la relación; el llamador debe abortar
// el arranque ante ese error.
func LoadRelaciones(dir string, urls ServiceURLs) (map[string]ServiceConfig, error) {
	patron := filepath.Join(dir, "relaciones_*.yaml")
	rutas, err := filepath.Glob(patron)
	if err != nil {
		return nil, fmt.Errorf("no se pudo listar archivos de relaciones en %q: %w", dir, err)
	}
	sort.Strings(rutas)

	configs := make(map[string]ServiceConfig, len(rutas))
	for _, ruta := range rutas {
		sc, err := cargarArchivoRelaciones(ruta)
		if err != nil {
			return nil, err
		}
		if _, duplicado := configs[sc.Service]; duplicado {
			return nil, fmt.Errorf(
				"configuración de relaciones inválida en %q: el servicio %q ya fue declarado en otro archivo",
				ruta, sc.Service,
			)
		}
		if err := validarServiceConfig(ruta, sc, urls); err != nil {
			return nil, err
		}
		configs[sc.Service] = sc
	}

	return configs, nil
}

func cargarArchivoRelaciones(ruta string) (ServiceConfig, error) {
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		return ServiceConfig{}, fmt.Errorf("no se pudo leer el archivo de relaciones %q: %w", ruta, err)
	}

	var sc ServiceConfig
	dec := yaml.NewDecoder(strings.NewReader(string(contenido)))
	dec.KnownFields(true)
	if err := dec.Decode(&sc); err != nil {
		return ServiceConfig{}, fmt.Errorf("archivo de relaciones inválido %q: %w", ruta, err)
	}

	return sc, nil
}

// validarServiceConfig aplica las reglas de validación de un servicio y aborta
// con un mensaje que identifica el archivo y la relación inconsistente.
func validarServiceConfig(ruta string, sc ServiceConfig, urls ServiceURLs) error {
	if strings.TrimSpace(sc.Service) == "" {
		return fmt.Errorf("configuración de relaciones inválida en %q: el campo service es obligatorio", ruta)
	}
	if sc.MaxDepth <= 0 {
		return fmt.Errorf("configuración de relaciones inválida en %q: max_depth debe ser positivo", ruta)
	}
	if sc.OnRelationError != OnRelationErrorNotify && sc.OnRelationError != OnRelationErrorFail {
		return fmt.Errorf(
			"configuración de relaciones inválida en %q: on_relation_error debe ser %q o %q",
			ruta, OnRelationErrorNotify, OnRelationErrorFail,
		)
	}
	if sc.MaxRelatedIDs <= 0 {
		return fmt.Errorf("configuración de relaciones inválida en %q: max_related_ids debe ser positivo", ruta)
	}

	// mapped_name único por servicio (en el nivel superior de relaciones).
	vistos := make(map[string]struct{}, len(sc.Relations))
	for i := range sc.Relations {
		rel := sc.Relations[i]
		if err := validarRelacion(ruta, sc.Service, rel, urls); err != nil {
			return err
		}
		if _, dup := vistos[rel.MappedName]; dup {
			return fmt.Errorf(
				"configuración de relaciones inválida en %q: mapped_name %q duplicado en el servicio %q",
				ruta, rel.MappedName, sc.Service,
			)
		}
		vistos[rel.MappedName] = struct{}{}
	}

	return nil
}

// validarRelacion valida una relación y recursivamente sus relaciones anidadas.
func validarRelacion(ruta, servicio string, rel RelationConfig, urls ServiceURLs) error {
	id := rel.Name
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf(
			"configuración de relaciones inválida en %q (servicio %q): el campo name de una relación es obligatorio",
			ruta, servicio,
		)
	}

	fallo := func(campo string) error {
		return fmt.Errorf(
			"configuración de relaciones inválida en %q (servicio %q, relación %q): %s",
			ruta, servicio, id, campo,
		)
	}

	for campo, valor := range map[string]string{
		"related_service": rel.RelatedService,
		"related_api":     rel.RelatedAPI,
		"related_table":   rel.RelatedTable,
		"local_field":     rel.LocalField,
		"related_field":   rel.RelatedField,
		"mapped_name":     rel.MappedName,
		"endpoint":        rel.Endpoint,
	} {
		if strings.TrimSpace(valor) == "" {
			return fallo(fmt.Sprintf("el campo %s es obligatorio", campo))
		}
	}

	if rel.Direction != DirectionLocalToRelated && rel.Direction != DirectionRelatedToLocal {
		return fallo(fmt.Sprintf("direction debe ser %q o %q", DirectionLocalToRelated, DirectionRelatedToLocal))
	}
	if rel.Cardinality != CardinalityOne && rel.Cardinality != CardinalityMany {
		return fallo(fmt.Sprintf("cardinality debe ser %q o %q", CardinalityOne, CardinalityMany))
	}
	if rel.Depth <= 0 {
		return fallo("depth debe ser positivo")
	}
	if _, existe := urls[rel.RelatedAPI]; !existe {
		return fallo(fmt.Sprintf("related_api %q no existe en la configuración de URLs", rel.RelatedAPI))
	}

	// mapped_name único entre las relaciones hermanas anidadas.
	vistos := make(map[string]struct{}, len(rel.NestedRelations))
	for i := range rel.NestedRelations {
		hija := rel.NestedRelations[i]
		if err := validarRelacion(ruta, servicio, hija, urls); err != nil {
			return err
		}
		if _, dup := vistos[hija.MappedName]; dup {
			return fmt.Errorf(
				"configuración de relaciones inválida en %q (servicio %q, relación %q): mapped_name %q duplicado entre relaciones anidadas",
				ruta, servicio, id, hija.MappedName,
			)
		}
		vistos[hija.MappedName] = struct{}{}
	}

	return nil
}

// MustLoadRelaciones carga y valida los archivos de relaciones y aborta el
// arranque del proceso si la configuración está ausente, es inválida o es
// inconsistente, imprimiendo el mensaje que identifica el archivo y la relación.
func MustLoadRelaciones(dir string, urls ServiceURLs) map[string]ServiceConfig {
	configs, err := LoadRelaciones(dir, urls)
	if err != nil {
		log.Fatalf("no se pudo cargar la configuración de relaciones: %v", err)
	}
	return configs
}
