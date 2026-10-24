package services

import (
	"context"
	"errors"

	"github.com/udistrital/agora_mid/clients"
	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/helper"
)

// ErrRamaDetalleIncoherente indica que el cuerpo de escritura declara una rama de
// detalle (ProveedorNatural/ProveedorJuridico) que no corresponde al tipo de
// registro del proveedor, o declara ambas a la vez. El controlador debe mapearlo
// a 400 Bad Request. Es una regla de negocio irreductible que no vive en el motor
// genérico sino en el servicio del proveedor.
var ErrRamaDetalleIncoherente = errors.New(
	"la rama de detalle del cuerpo no corresponde al tipo de registro del proveedor",
)

// ProveedorWriteService orquesta la escritura del proveedor como recurso raíz:
// aplica las reglas de negocio irreductibles (coherencia natural/jurídico), poda
// del árbol la rama de detalle que no corresponde al tipo de registro e invoca el
// motor de escritura genérico con el registro de write-resolvers del núcleo.
//
// Las reglas estructurales (orden topológico, propagación de llave, aplanado,
// idempotencia, inmutables) viven en el motor y la configuración; aquí solo vive
// lo que el motor no puede expresar de forma declarativa.
type ProveedorWriteService struct {
	// Config es la configuración declarativa de relaciones del proveedor, con su
	// bloque write.
	Config config.ServiceConfig
	// Registry es el registro de write-resolvers que el servicio inyecta al motor.
	Registry helper.WriteResolverRegistry
	// TipoRegistroNatural es el identificador de tipo de contribuyente que
	// corresponde a persona natural. Determina qué rama de detalle es válida.
	TipoRegistroNatural int
}

// NewProveedorWriteService construye el servicio de escritura con la
// configuración de relaciones del proveedor y el registro de write-resolvers del
// núcleo resuelto contra las URLs de los CRUD.
func NewProveedorWriteService(
	client clients.HTTPClient,
	urls config.ServiceURLs,
	cfg config.ServiceConfig,
	tipoRegistroNatural int,
) ProveedorWriteService {
	raiz, hijos := NewWriteResolversNucleo(client, urls)
	registry := helper.NewWriteResolverRegistry(raiz, hijos)
	return ProveedorWriteService{
		Config:              cfg,
		Registry:            registry,
		TipoRegistroNatural: tipoRegistroNatural,
	}
}

// EscribirProveedor materializa el subárbol del proveedor a partir del cuerpo
// anidado. verb es WriteVerbCreate (POST) o WriteVerbUpdate (PATCH). Aplica la
// coherencia natural/jurídico, poda la rama de detalle que no corresponde y
// delega la cascada en el motor de escritura.
//
// Devuelve ErrRamaDetalleIncoherente o *helper.ErrorEscritura (ambos 400) ante un
// cuerpo inválido, un error fatal de escritura (ver helper.EsErrorFatalEscritura)
// cuando la política de relación es fail, o el WriteResult con los identificadores
// materializados en el caso tolerante.
func (s ProveedorWriteService) EscribirProveedor(
	ctx context.Context,
	body map[string]any,
	verb string,
) (helper.WriteResult, error) {
	if body == nil {
		return helper.WriteResult{}, &helper.ErrorEscritura{
			Path: "(raíz)", Motivo: "el cuerpo de escritura es obligatorio",
		}
	}

	tipoRegistro := idEntero(body["TipoRegistroId"])
	if tipoRegistro <= 0 {
		return helper.WriteResult{}, &helper.ErrorEscritura{
			Path: "TipoRegistroId", Motivo: "es obligatorio y debe ser un entero positivo",
		}
	}

	// Regla irreductible: la rama de detalle debe corresponder al tipo de
	// registro, y no pueden declararse ambas ramas a la vez.
	if err := s.validarCoherenciaRama(body, tipoRegistro); err != nil {
		return helper.WriteResult{}, err
	}

	// Podar del árbol la rama de detalle que no corresponde al tipo de registro,
	// igual que en lectura; la rama ausente del cuerpo tampoco se recorre.
	cfg := s.configParaTipo(tipoRegistro)

	return helper.Write(ctx, body, cfg, s.Registry, helper.WriteOptions{Verb: verb})
}

// validarCoherenciaRama verifica que el cuerpo declare únicamente la rama de
// detalle que corresponde al tipo de registro. Un proveedor natural no puede
// traer ProveedorJuridico y viceversa, ni ambas a la vez.
func (s ProveedorWriteService) validarCoherenciaRama(body map[string]any, tipoRegistro int) error {
	_, traeNatural := body[relacionProveedorNatural]
	_, traeJuridico := body[relacionProveedorJuridico]

	if traeNatural && traeJuridico {
		return ErrRamaDetalleIncoherente
	}

	esNatural := tipoRegistro == s.TipoRegistroNatural
	if esNatural && traeJuridico {
		return ErrRamaDetalleIncoherente
	}
	if !esNatural && traeNatural {
		return ErrRamaDetalleIncoherente
	}
	return nil
}

// configParaTipo devuelve la configuración con la rama de detalle que no
// corresponde al tipo de registro podada, de modo que el motor no intente
// materializar la rama incorrecta.
func (s ProveedorWriteService) configParaTipo(tipoRegistro int) config.ServiceConfig {
	descartar := relacionProveedorJuridico
	if tipoRegistro != s.TipoRegistroNatural {
		descartar = relacionProveedorNatural
	}

	cfg := s.Config
	relaciones := make([]config.RelationConfig, 0, len(cfg.Relations))
	for _, rel := range cfg.Relations {
		if rel.MappedName == descartar {
			continue
		}
		relaciones = append(relaciones, rel)
	}
	cfg.Relations = relaciones
	return cfg
}
