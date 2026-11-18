package generador

import (
	"fmt"
	"path/filepath"
)

// ConstruirServicios genera un archivo de servicio por cada entrada del sidecar
// service. Cada archivo recibe las aristas que su raíz tiene en el mapa de objetos, la
// política de escritura de la raíz (si write la declara) y los parámetros del sidecar.
// Devuelve error si la raíz pertenece a un esquema sin CRUD mapeado.
func ConstruirServicios(m *Modelo, s *Sidecars) ([]ServicioConfig, error) {
	_, servicios, err := construirTodo(m, s)
	return servicios, err
}

// serviciosDesde arma un archivo de servicio por raíz a partir del mapa de objetos. Las
// aristas se copian para que no dependan de si la raíz ya fue vaciada en objetosDesde.
func serviciosDesde(mapa map[string]*ObjetoConfig, s *Sidecars) ([]ServicioConfig, error) {
	servicios := []ServicioConfig{}
	for _, raiz := range clavesOrdenadas(s.Service) {
		o, ok := mapa[raiz]
		if !ok {
			return nil, fmt.Errorf("sidecar %s: la raíz %q no pertenece a un CRUD mapeado", archivoService, raiz)
		}
		sv := s.Service[raiz]
		campos := make([]string, 0, len(sv.FilterableFields))
		for _, f := range sv.FilterableFields {
			campos = append(campos, pascal(f))
		}
		svc := ServicioConfig{
			Service:          sv.Service,
			Raiz:             o.Name,
			MaxDepth:         sv.MaxDepth,
			OnRelationError:  sv.OnRelationError,
			MaxRelatedIDs:    sv.MaxRelatedIDs,
			FilterableFields: campos,
			Referencias:      append([]AristaConfig{}, o.Referencias...),
			RelatedAPI:       o.RelatedAPI,
			RelatedTable:     o.RelatedTable,
		}
		if pol, ok := s.Write[raiz]; ok {
			svc.Write = &EscrituraServicio{Policy: politica(pol)}
		}
		servicios = append(servicios, svc)
	}
	return servicios, nil
}

// EscribirServicios escribe cada archivo de servicio en
// salida/<related_api>/relaciones_<service>.yaml, creando las carpetas necesarias. La
// serialización sigue el orden de los campos de ServicioConfig, así que la misma
// entrada produce los mismos bytes.
func EscribirServicios(servicios []ServicioConfig, salida string) error {
	for _, svc := range servicios {
		if err := escribirYAML(filepath.Join(salida, svc.RelatedAPI), "relaciones_"+svc.Service+".yaml", svc); err != nil {
			return err
		}
	}
	return nil
}
