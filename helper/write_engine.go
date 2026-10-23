package helper

import (
	"context"
	"fmt"
	"strings"

	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/responses"
)

// Verbos de escritura reconocidos por el motor. Determinan qué política por
// verbo de cada relación habilita la materialización.
const (
	// WriteVerbCreate corresponde a una creación del subárbol (POST).
	WriteVerbCreate = "create"
	// WriteVerbUpdate corresponde a una reconciliación del subárbol (PATCH).
	WriteVerbUpdate = "update"
)

// Sufijo que el motor usa al aplanar un objeto de parámetro anidado a su llave
// foránea: {X:{Id:n}} -> XId:n.
const sufijoFK = "Id"

// ErrorEscritura representa un error de validación del cuerpo de escritura que el
// llamador debe mapear a 400 Bad Request. Identifica la ruta lógica inválida sin
// exponer datos sensibles. Es el análogo de escritura de ErrorFields.
type ErrorEscritura struct {
	// Path es la ruta lógica del punto inválido (por ejemplo "Hijo.Campo").
	Path string
	// Motivo describe la causa sin exponer valores del cuerpo.
	Motivo string
}

func (e *ErrorEscritura) Error() string {
	return fmt.Sprintf("cuerpo de escritura inválido en %q: %s", e.Path, e.Motivo)
}

// WriteResolver materializa una entidad contra su CRUD dueño y devuelve el
// registro materializado, que debe incluir su identificador. keys contiene los
// valores de la clave natural declarada, si la hay, para que el resolver decida
// si reutiliza un registro existente (upsert idempotente) o crea uno nuevo. Un
// keys vacío indica que la idempotencia se apoya en el Id presente en payload.
//
// El motor no construye URLs ni nombra CRUD: delega por completo el origen real
// de cada entidad en el WriteResolver, de forma simétrica a Resolver en lectura.
type WriteResolver interface {
	Materialize(
		ctx context.Context,
		rel config.RelationConfig,
		payload map[string]any,
		keys []string,
	) (map[string]any, error)
}

// WriteResolverRegistry asocia un related_service con el WriteResolver que lo
// materializa, con un resolver por defecto para la entidad raíz (que no es una
// relación y por tanto no tiene related_service).
type WriteResolverRegistry struct {
	porServicio map[string]WriteResolver
	raiz        WriteResolver
}

// NewWriteResolverRegistry construye un registro con el resolver de la entidad
// raíz y los resolvers especializados por related_service.
func NewWriteResolverRegistry(raiz WriteResolver, especializados map[string]WriteResolver) WriteResolverRegistry {
	cp := make(map[string]WriteResolver, len(especializados))
	for k, v := range especializados {
		cp[k] = v
	}
	return WriteResolverRegistry{porServicio: cp, raiz: raiz}
}

// resolverRaiz devuelve el resolver de la entidad raíz.
func (r WriteResolverRegistry) resolverRaiz() WriteResolver { return r.raiz }

// resolverPara devuelve el resolver de una relación por su related_service.
func (r WriteResolverRegistry) resolverPara(rel config.RelationConfig) (WriteResolver, bool) {
	res, ok := r.porServicio[rel.RelatedService]
	return res, ok
}

// WriteOptions son las opciones de una operación de escritura.
type WriteOptions struct {
	// Verb es el verbo de escritura (WriteVerbCreate o WriteVerbUpdate). Vacío se
	// interpreta como creación.
	Verb string
	// OnRelationError es la política ante el fallo de una relación: notify
	// (tolerante) o fail (estricto). Vacío toma la de la ServiceConfig.
	OnRelationError string
}

// WriteResult es el resultado de una escritura en cascada: el subárbol con los
// identificadores materializados y los avisos acumulados para MetaData.Errors.
type WriteResult struct {
	// Data es el subárbol materializado, con los Id de cada entidad creada o
	// reconciliada, para permitir el reintento idempotente.
	Data map[string]any
	// Warnings son los avisos de relaciones toleradas no materializadas.
	Warnings []responses.RelationError
}

// errorFatalEscritura envuelve un error que aborta toda la cascada de escritura
// en modo fail (o ante una relación bloqueante). El llamador debe mapearlo a
// 502/504.
type errorFatalEscritura struct {
	path     string
	servicio string
	err      error
}

func (e *errorFatalEscritura) Error() string {
	return fmt.Sprintf("escritura abortada en %q (servicio %q): error de relación", e.path, e.servicio)
}

func (e *errorFatalEscritura) Unwrap() error { return e.err }

// EsErrorFatalEscritura informa si err corresponde a un aborto de la cascada de
// escritura por modo fail o relación bloqueante.
func EsErrorFatalEscritura(err error) bool {
	_, ok := err.(*errorFatalEscritura)
	return ok
}

// NewErrorFatalEscritura envuelve err como un fallo bloqueante de escritura que
// el llamador debe mapear a 502/504. Conserva err en la cadena para que el
// controlador distinga un tiempo de espera agotado (504) de otros fallos (502).
func NewErrorFatalEscritura(err error) error {
	return &errorFatalEscritura{path: "(raíz)", err: err}
}

// motorEscritura mantiene el estado de una escritura en cascada: resolvers,
// verbo y política de error efectivos y avisos acumulados.
type motorEscritura struct {
	ctx        context.Context
	registry   WriteResolverRegistry
	verb       string
	onRelError string
	warnings   []responses.RelationError
}

// Write materializa un cuerpo anidado en cascada a partir de la configuración de
// escritura del servicio. Recorre el árbol en orden topológico (padre antes que
// hijos), propaga la llave del padre a cada hijo según el modo declarado, aplana
// los parámetros anidados a llave foránea y delega la materialización de cada
// entidad en su WriteResolver.
//
// Validación previa sin materializar: antes de cualquier escritura, el cuerpo
// completo se valida y aplana contra las políticas de escritura declaradas; un
// campo no escribible o un intento de modificar un inmutable devuelve
// *ErrorEscritura (que el llamador mapea a 400) sin materializar nada.
//
// Semántica de fallo espejo de la lectura: en modo notify una relación no
// bloqueante que falla conserva lo ya materializado y agrega un aviso no
// sensible; en modo fail (o relación bloqueante) la cascada se aborta con
// *errorFatalEscritura.
//
// Devuelve *ErrorEscritura ante un cuerpo inválido, *errorFatalEscritura ante un
// aborto por política estricta y, en el caso tolerante, un WriteResult con los
// identificadores materializados y los avisos.
func Write(
	ctx context.Context,
	body map[string]any,
	cfg config.ServiceConfig,
	registry WriteResolverRegistry,
	options WriteOptions,
) (WriteResult, error) {
	if cfg.Write == nil {
		return WriteResult{}, fmt.Errorf("write: el servicio %q no declara configuración de escritura", cfg.Service)
	}

	verb := strings.TrimSpace(options.Verb)
	if verb == "" {
		verb = WriteVerbCreate
	}
	if verb != WriteVerbCreate && verb != WriteVerbUpdate {
		return WriteResult{}, fmt.Errorf("write: verbo inválido %q", verb)
	}

	onRelError := strings.TrimSpace(options.OnRelationError)
	if onRelError == "" {
		onRelError = cfg.OnRelationError
	}
	if onRelError != config.OnRelationErrorNotify && onRelError != config.OnRelationErrorFail {
		return WriteResult{}, fmt.Errorf("write: on_relation_error inválido %q", onRelError)
	}

	// Validación y aplanado de todo el árbol antes de materializar. Un cuerpo
	// inválido se rechaza con *ErrorEscritura sin ninguna escritura.
	if err := validarNivel(body, *cfg.Write, cfg.Relations, verb, ""); err != nil {
		return WriteResult{}, err
	}

	me := &motorEscritura{
		ctx:        ctx,
		registry:   registry,
		verb:       verb,
		onRelError: onRelError,
	}

	// Pre-pass de FK ascendente (fk_inbound): materializar las entidades que el
	// nodo raíz referencia por una FK local (por ejemplo tercero, referenciado por
	// proveedor.TerceroId) ANTES de la raíz, y propagar su Id hacia arriba a la FK
	// local de la raíz. Así la raíz se crea con su FK ya resuelta.
	if err := me.materializarInbound(body, cfg.Relations, nil); err != nil {
		return WriteResult{}, err
	}

	// Materializar la entidad raíz y luego su subárbol en orden topológico. Los
	// campos materializados (en particular el Id) se fusionan en el cuerpo
	// original, que conserva los valores de las relaciones hijas para el recorrido.
	materializado, err := me.materializarEntidad(
		registry.resolverRaiz(), config.RelationConfig{}, body, *cfg.Write, "(raíz)",
	)
	if err != nil {
		return WriteResult{}, &errorFatalEscritura{path: "(raíz)", servicio: cfg.Service, err: err}
	}
	for k, v := range materializado {
		body[k] = v
	}

	if err := me.materializarHijos(body, cfg.Relations, nil); err != nil {
		return WriteResult{}, err
	}

	return WriteResult{Data: body, Warnings: me.warnings}, nil
}

// validarNivel valida y aplana recursivamente un nivel del cuerpo contra la
// configuración de escritura de la entidad y las relaciones hijas, sin
// materializar. Rechaza con *ErrorEscritura un campo no escribible, un intento de
// modificar un inmutable en actualización y una relación no declarada escribible.
func validarNivel(
	entidad map[string]any,
	wc config.WriteConfig,
	relaciones []config.RelationConfig,
	verb string,
	ruta string,
) error {
	escribibles := conjunto(wc.WritableFields)
	inmutables := conjunto(wc.ImmutableFields)
	porMapped := indexarPorMapped(relaciones)

	for clave, valor := range entidad {
		rutaCampo := unirRuta(ruta, clave)

		// Un campo que corresponde a una relación hija escribible se valida como
		// subárbol, no como campo propio.
		if rel, esRel := porMapped[clave]; esRel {
			if rel.Write == nil {
				return &ErrorEscritura{Path: rutaCampo, Motivo: "la relación no admite escritura"}
			}
			if err := validarRelacionAnidada(valor, *rel.Write, rel.NestedRelations, verb, rutaCampo); err != nil {
				return err
			}
			continue
		}

		// Un objeto de parámetro anidado {Id:n} se aplana a la FK <clave>Id y la
		// FK resultante debe ser escribible.
		if esParametroAnidado(valor) {
			campoFK := clave + sufijoFK
			if !contiene(escribibles, campoFK) {
				return &ErrorEscritura{
					Path:   rutaCampo,
					Motivo: fmt.Sprintf("el parámetro anidado no corresponde a un campo escribible (%s)", campoFK),
				}
			}
			continue
		}

		// El Id se acepta como identificador de la entidad para el upsert, no como
		// campo escribible.
		if clave == "Id" {
			continue
		}

		if !contiene(escribibles, clave) {
			return &ErrorEscritura{Path: rutaCampo, Motivo: "campo no escribible"}
		}
		if verb == WriteVerbUpdate && contiene(inmutables, clave) {
			return &ErrorEscritura{Path: rutaCampo, Motivo: "campo inmutable; no puede modificarse"}
		}
	}

	return nil
}

// validarRelacionAnidada valida el valor de una relación hija, que puede ser un
// objeto único (cardinalidad 1-1) o una lista (cardinalidad 1-N).
func validarRelacionAnidada(
	valor any,
	wc config.WriteConfig,
	nested []config.RelationConfig,
	verb string,
	ruta string,
) error {
	switch v := valor.(type) {
	case map[string]any:
		return validarNivel(v, wc, nested, verb, ruta)
	case []any:
		for i := range v {
			elem, ok := v[i].(map[string]any)
			if !ok {
				return &ErrorEscritura{
					Path:   fmt.Sprintf("%s[%d]", ruta, i),
					Motivo: "cada elemento de la colección debe ser un objeto",
				}
			}
			if err := validarNivel(elem, wc, nested, verb, fmt.Sprintf("%s[%d]", ruta, i)); err != nil {
				return err
			}
		}
		return nil
	case nil:
		return nil
	default:
		return &ErrorEscritura{Path: ruta, Motivo: "la relación debe ser un objeto o una lista de objetos"}
	}
}

// materializarHijos recorre las relaciones hijas de un padre ya materializado,
// propaga la llave del padre según el modo declarado y materializa cada hijo en
// orden topológico.
func (me *motorEscritura) materializarHijos(
	padre map[string]any,
	relaciones []config.RelationConfig,
	rutaRama []string,
) error {
	// El padre ya fue materializado: su Id es la llave que se propaga a cada hijo.
	padreID := ""
	if v, ok := padre["Id"]; ok && v != nil {
		padreID = aCadena(v)
	}

	for i := range relaciones {
		rel := relaciones[i]
		if rel.Write == nil {
			continue
		}
		// Las relaciones fk_inbound se materializan en el pre-pass (antes del
		// nodo), no en el recorrido descendente padre->hijo.
		if rel.Write.KeyPropagation == config.KeyPropagationFKInbound {
			continue
		}

		valor, presente := padre[rel.MappedName]
		if !presente || valor == nil {
			continue
		}

		rutaActual := append(append([]string{}, rutaRama...), rel.MappedName)
		rutaStr := strings.Join(rutaActual, ".")

		res, ok := me.registry.resolverPara(rel)
		if !ok {
			if err := me.manejarErrorRelacion(rel, rutaStr); err != nil {
				return err
			}
			continue
		}

		if err := me.materializarValorRelacion(valor, rel, res, padreID, rutaActual, rutaStr); err != nil {
			return err
		}
	}
	return nil
}

// materializarInbound ejecuta el pre-pass de FK ascendente (fk_inbound): por cada
// relación del nodo declarada fk_inbound y presente en el cuerpo, materializa la
// entidad referenciada (y su propio subárbol) ANTES que el nodo y propaga su Id
// hacia arriba, a la FK local del nodo (rel.LocalField). La entidad referenciada
// materializada queda anidada en el nodo bajo su mapped_name para la respuesta.
//
// Solo aplica a relaciones de cardinalidad 1-1: una FK local del nodo apunta a un
// único registro referenciado.
func (me *motorEscritura) materializarInbound(
	nodo map[string]any,
	relaciones []config.RelationConfig,
	rutaRama []string,
) error {
	for i := range relaciones {
		rel := relaciones[i]
		if rel.Write == nil || rel.Write.KeyPropagation != config.KeyPropagationFKInbound {
			continue
		}

		valor, presente := nodo[rel.MappedName]
		if !presente || valor == nil {
			continue
		}
		referida, ok := valor.(map[string]any)
		if !ok {
			return &ErrorEscritura{
				Path:   strings.Join(append(append([]string{}, rutaRama...), rel.MappedName), "."),
				Motivo: "una relación fk_inbound debe ser un objeto único",
			}
		}

		rutaActual := append(append([]string{}, rutaRama...), rel.MappedName)
		rutaStr := strings.Join(rutaActual, ".")

		res, ok := me.registry.resolverPara(rel)
		if !ok {
			if err := me.manejarErrorRelacion(rel, rutaStr); err != nil {
				return err
			}
			continue
		}

		materializado, err := me.materializarEntidad(res, rel, referida, *rel.Write, rutaStr)
		if err != nil {
			return me.manejarErrorRelacion(rel, rutaStr)
		}
		for k, v := range materializado {
			referida[k] = v
		}

		// Propagar hacia arriba: la FK local del nodo toma el Id de la entidad
		// referenciada recién materializada.
		if id, hay := referida["Id"]; hay && id != nil {
			nodo[rel.LocalField] = aCadena(id)
		}

		// Materializar el subárbol propio de la entidad referenciada (por ejemplo
		// los datos_identificacion del tercero).
		if err := me.materializarHijos(referida, rel.NestedRelations, rutaActual); err != nil {
			return err
		}
	}
	return nil
}

// materializarValorRelacion materializa el valor de una relación, que puede ser
// un objeto (1-1) o una lista (1-N), propagando la llave del padre a cada
// elemento antes de materializarlo.
func (me *motorEscritura) materializarValorRelacion(
	valor any,
	rel config.RelationConfig,
	res WriteResolver,
	padreID string,
	rutaActual []string,
	rutaStr string,
) error {
	procesar := func(elem map[string]any, ruta string) error {
		propagarLlave(elem, rel, padreID)
		materializado, err := me.materializarEntidad(res, rel, elem, *rel.Write, ruta)
		if err != nil {
			return me.manejarErrorRelacion(rel, ruta)
		}
		// Reemplazar el elemento por su versión materializada y recorrer sus
		// propios hijos en orden topológico.
		for k, v := range materializado {
			elem[k] = v
		}
		return me.materializarHijos(elem, rel.NestedRelations, rutaActual)
	}

	switch v := valor.(type) {
	case map[string]any:
		return procesar(v, rutaStr)
	case []any:
		for i := range v {
			elem, ok := v[i].(map[string]any)
			if !ok {
				continue
			}
			if err := procesar(elem, fmt.Sprintf("%s[%d]", rutaStr, i)); err != nil {
				return err
			}
		}
		return nil
	default:
		return nil
	}
}

// materializarEntidad aplana los parámetros anidados del payload y delega la
// materialización en el resolver, pasando los valores de la clave natural
// declarada para el upsert idempotente.
func (me *motorEscritura) materializarEntidad(
	res WriteResolver,
	rel config.RelationConfig,
	payload map[string]any,
	wc config.WriteConfig,
	ruta string,
) (map[string]any, error) {
	if res == nil {
		return nil, fmt.Errorf("write: no hay resolver para %q", ruta)
	}
	plano := aplanarParametros(payload, wc.WritableFields, rel.NestedRelations)
	keys := valoresClaveNatural(plano, wc.NaturalKey)
	return res.Materialize(me.ctx, rel, plano, keys)
}

// propagarLlave fija en el hijo la llave del padre según el modo declarado:
//   - fk: fija la llave foránea del hijo (related_field) con el Id del padre.
//   - shared_pk: fija el propio Id del hijo con el Id del padre.
func propagarLlave(hijo map[string]any, rel config.RelationConfig, padreID string) {
	if padreID == "" {
		return
	}
	switch rel.Write.KeyPropagation {
	case config.KeyPropagationFK:
		hijo[rel.RelatedField] = padreID
	case config.KeyPropagationSharedPK:
		hijo["Id"] = padreID
	}
}

// manejarErrorRelacion aplica la política de error de relación. En modo fail
// aborta con *errorFatalEscritura conservando lo ya materializado implícitamente
// en el cuerpo del llamador; en modo notify agrega un aviso no sensible y
// continúa.
func (me *motorEscritura) manejarErrorRelacion(rel config.RelationConfig, rutaStr string) error {
	if me.onRelError == config.OnRelationErrorFail {
		return &errorFatalEscritura{path: rutaStr, servicio: rel.RelatedService}
	}
	me.agregarAviso(responses.RelationError{
		Path:    rutaStr,
		Service: rel.RelatedService,
		Message: "no se pudo materializar la relación; se conserva lo ya materializado",
		Kind:    KindRelation,
	})
	return nil
}

// agregarAviso añade un aviso si no está duplicado por path+kind.
func (me *motorEscritura) agregarAviso(aviso responses.RelationError) {
	for _, a := range me.warnings {
		if a.Path == aviso.Path && a.Kind == aviso.Kind {
			return
		}
	}
	me.warnings = append(me.warnings, aviso)
}

// aplanarParametros devuelve una copia del payload con los objetos de parámetro
// anidados ({X:{Id:n}}) aplanados a su llave foránea (XId:n) y sin las claves de
// relaciones hijas (que se materializan por separado). Solo conserva los campos
// escribibles declarados, el Id y las FK aplanadas.
func aplanarParametros(
	payload map[string]any,
	writable []string,
	nested []config.RelationConfig,
) map[string]any {
	escribibles := conjunto(writable)
	porMapped := indexarPorMapped(nested)
	salida := make(map[string]any, len(payload))

	for clave, valor := range payload {
		// Omitir las relaciones hijas: se materializan en su propia pasada.
		if _, esRel := porMapped[clave]; esRel {
			continue
		}
		if esParametroAnidado(valor) {
			campoFK := clave + sufijoFK
			if contiene(escribibles, campoFK) {
				salida[campoFK] = idDeParametro(valor)
			}
			continue
		}
		if clave == "Id" || contiene(escribibles, clave) {
			salida[clave] = valor
		}
	}
	return salida
}

// valoresClaveNatural extrae del payload aplanado los valores de la clave natural
// declarada, en el orden de los campos, normalizados a string. Devuelve nil si la
// relación no declara clave natural.
func valoresClaveNatural(plano map[string]any, naturalKey []string) []string {
	if len(naturalKey) == 0 {
		return nil
	}
	keys := make([]string, 0, len(naturalKey))
	for _, campo := range naturalKey {
		if v, ok := plano[campo]; ok && v != nil {
			keys = append(keys, aCadena(v))
		}
	}
	return keys
}

// esParametroAnidado informa si el valor es un objeto de parámetro anidado, es
// decir, un map que solo aporta un Id (se aplana a FK). Un map con más campos se
// trata como una relación hija y no como un parámetro.
func esParametroAnidado(valor any) bool {
	m, ok := valor.(map[string]any)
	if !ok {
		return false
	}
	if _, tieneID := m["Id"]; !tieneID {
		return false
	}
	return len(m) == 1
}

// idDeParametro devuelve el Id de un objeto de parámetro anidado.
func idDeParametro(valor any) any {
	m, ok := valor.(map[string]any)
	if !ok {
		return nil
	}
	return m["Id"]
}

// indexarPorMapped indexa las relaciones por su mapped_name para una búsqueda
// rápida.
func indexarPorMapped(relaciones []config.RelationConfig) map[string]config.RelationConfig {
	idx := make(map[string]config.RelationConfig, len(relaciones))
	for i := range relaciones {
		idx[relaciones[i].MappedName] = relaciones[i]
	}
	return idx
}

// conjunto construye un conjunto de strings.
func conjunto(valores []string) map[string]struct{} {
	s := make(map[string]struct{}, len(valores))
	for _, v := range valores {
		s[v] = struct{}{}
	}
	return s
}

// contiene informa si el conjunto contiene la clave.
func contiene(s map[string]struct{}, clave string) bool {
	_, ok := s[clave]
	return ok
}

// unirRuta une una ruta base con un segmento usando el punto como separador.
func unirRuta(base, seg string) string {
	if base == "" {
		return seg
	}
	return base + "." + seg
}
