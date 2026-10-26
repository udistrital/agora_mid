package helper

import (
	"context"
	"fmt"
	"strings"

	"github.com/udistrital/agora_mid/config"
	"github.com/udistrital/agora_mid/responses"
)

// Tipos de aviso de expansión serializados en MetaData.Errors. Todo Kind emitido
// por el motor de expansión corresponde a una de estas constantes; no debe
// emitirse ningún Kind fuera de este conjunto.
const (
	// KindRelation marca un error de relación tolerado en modo notify.
	KindRelation = "relation"
	// KindDepth marca la reducción de profundidad por configuración hija.
	KindDepth = "depth"
	// KindCycle marca la detención de una rama por ciclo detectado.
	KindCycle = "cycle"
	// KindGenericResolver marca una relación resuelta por el resolver genérico,
	// cuyo Data se entrega sin transformar.
	KindGenericResolver = "generic_resolver"
)

// KindsDeAviso enumera el conjunto completo de tipos de aviso declarados. El
// invariante del motor es que todo Kind emitido pertenece a este conjunto.
var KindsDeAviso = []string{
	KindRelation,
	KindDepth,
	KindCycle,
	KindGenericResolver,
}

// ExpandOptions son las opciones de expansión que arma el controlador. El helper
// no implementa recursión en el controlador: recibe estas opciones, resuelve y
// devuelve el resultado.
type ExpandOptions struct {
	// Fields son las rutas de selección (por ejemplo Id, Tercero.Id). Vacío
	// expande todas las relaciones permitidas hasta la profundidad efectiva.
	Fields []string
	// MaxDepth es el límite global de profundidad. Si es <= 0 se toma el de la
	// ServiceConfig.
	MaxDepth int
	// MaxRelatedIDs limita las llaves consultadas por lote. Si es <= 0 se toma
	// el de la ServiceConfig.
	MaxRelatedIDs int
	// OnRelationError es la política de error de relación: notify o fail. Vacío
	// toma la de la ServiceConfig.
	OnRelationError string
	// Seleccion es un árbol de selección ya parseado y validado por el llamador.
	// Cuando no es nil, Expand lo usa directamente y no vuelve a parsear Fields,
	// de modo que la validación de fields tiene una única fuente de verdad. Si es
	// nil, Expand parsea Fields contra la configuración recibida.
	Seleccion *NodoSeleccion
	// Blacklist es la lista negra global de campos. Cuando no es nil, el motor
	// recorta los campos vetados de la salida de todo resolver (específico y
	// genérico) antes de anidar cada registro, con precedencia sobre allowed_fields
	// y sobre el Id implícito. Una lista negra nil o vacía no recorta nada.
	Blacklist config.GlobalBlacklist
}

// ExpandResult es el resultado de una expansión: los datos expandidos y los
// avisos acumulados para MetaData.Errors.
type ExpandResult struct {
	Data     any
	Warnings []responses.RelationError
}

// ResolverRegistry asocia un related_service con el Resolver que lo atiende.
// Cuando no existe un resolver especializado para una relación declarada, se usa
// Generic (envuelto como genérico) y se emite un aviso generic_resolver.
type ResolverRegistry struct {
	porServicio map[string]Resolver
	generic     Resolver
}

// NewResolverRegistry construye un registro con los resolvers especializados y el
// resolver genérico de respaldo.
func NewResolverRegistry(especializados map[string]Resolver, generic Resolver) ResolverRegistry {
	cp := make(map[string]Resolver, len(especializados))
	for k, v := range especializados {
		cp[k] = v
	}
	return ResolverRegistry{porServicio: cp, generic: generic}
}

// resolverPara devuelve el resolver para la relación y si es genérico.
func (r ResolverRegistry) resolverPara(rel config.RelationConfig) (Resolver, bool) {
	if res, ok := r.porServicio[rel.RelatedService]; ok {
		return res, EsGenerico(res)
	}
	if r.generic != nil {
		return r.generic, true
	}
	return nil, false
}

// errorFatalExpansion envuelve un error que aborta toda la expansión en modo
// fail. El llamador debe mapearlo a 502/504.
type errorFatalExpansion struct {
	path    string
	servicio string
	err     error
}

func (e *errorFatalExpansion) Error() string {
	return fmt.Sprintf("expansión abortada en %q (servicio %q): error de relación", e.path, e.servicio)
}

func (e *errorFatalExpansion) Unwrap() error { return e.err }

// EsErrorFatalExpansion informa si err corresponde a un aborto por modo fail.
func EsErrorFatalExpansion(err error) bool {
	_, ok := err.(*errorFatalExpansion)
	return ok
}

// expansor mantiene el estado de una expansión: opciones efectivas, resolvers,
// caché por solicitud y avisos acumulados.
type expansor struct {
	ctx        context.Context
	registry   ResolverRegistry
	maxDepth   int
	maxRelIDs  int
	onRelError string
	// blacklist es la lista negra global de campos; nil o vacía no recorta nada.
	blacklist config.GlobalBlacklist
	// cache por solicitud: clave "servicio|mapped|field|key" -> registros.
	cache    map[string][]map[string]any
	warnings []responses.RelationError
}

// Expand expande las relaciones de data según la configuración del servicio, las
// opciones y los resolvers del registro. data puede ser un map[string]any (un
// recurso) o []any / []map[string]any (una lista de recursos).
//
// Comportamiento:
//   - cardinality one sin resultado -> mapped_name = null.
//   - cardinality many sin resultado -> mapped_name = [].
//   - conserva siempre la llave local original.
//   - deduplica llaves, consulta por lotes y cachea por solicitud.
//   - profundidad efectiva = min(límite global restante, depth de la relación);
//     una config hija con menor profundidad agrega un aviso depth.
//   - detecta ciclos por mapped_name en la rama y agrega un aviso cycle.
//   - on_relation_error notify conserva el principal + aviso relation; fail
//     aborta con *errorFatalExpansion.
//
// Devuelve error (no fatal de relación) ante entradas o configuración inválidas,
// y *errorFatalExpansion cuando la política fail aborta la expansión.
func Expand(
	ctx context.Context,
	data any,
	cfg config.ServiceConfig,
	registry ResolverRegistry,
	options ExpandOptions,
) (ExpandResult, error) {
	maxDepth := options.MaxDepth
	if maxDepth <= 0 {
		maxDepth = cfg.MaxDepth
	}
	maxRelIDs := options.MaxRelatedIDs
	if maxRelIDs <= 0 {
		maxRelIDs = cfg.MaxRelatedIDs
	}
	onRelError := strings.TrimSpace(options.OnRelationError)
	if onRelError == "" {
		onRelError = cfg.OnRelationError
	}
	if onRelError != config.OnRelationErrorNotify && onRelError != config.OnRelationErrorFail {
		return ExpandResult{}, fmt.Errorf("expand: on_relation_error inválido %q", onRelError)
	}

	// Construir el árbol de selección a partir de fields (puede ser nil si no se
	// solicitaron fields: expansión completa). Si el llamador ya proveyó un árbol
	// parseado (options.Seleccion), se reutiliza para no parsear dos veces y
	// mantener una única fuente de verdad en la validación de fields.
	var seleccion *NodoSeleccion
	switch {
	case options.Seleccion != nil:
		seleccion = options.Seleccion
	case len(options.Fields) > 0:
		arbol, err := ParsearFields(options.Fields, cfg.Relations, allowedTopLevel(cfg), options.Blacklist)
		if err != nil {
			return ExpandResult{}, err
		}
		seleccion = arbol
	}

	ex := &expansor{
		ctx:        ctx,
		registry:   registry,
		maxDepth:   maxDepth,
		maxRelIDs:  maxRelIDs,
		onRelError: onRelError,
		blacklist:  options.Blacklist,
		cache:      make(map[string][]map[string]any),
	}

	salida, err := ex.expandirValor(data, cfg.Relations, seleccion, maxDepth, nil)
	if err != nil {
		return ExpandResult{}, err
	}

	return ExpandResult{Data: salida, Warnings: ex.warnings}, nil
}

// allowedTopLevel devuelve los allowed_fields del nivel base. El nivel base no
// declara allowed_fields a nivel de servicio; los campos base provienen del
// esquema del recurso. Se devuelve nil, lo que hace que ParsearFields acepte los
// campos base de nivel superior (su materialización/recorte no es parte de la
// expansión) y valide estrictamente las relaciones y sus descendientes contra la
// configuración jerárquica.
func allowedTopLevel(_ config.ServiceConfig) []string {
	return nil
}

// comoRecursos normaliza data a una lista de recursos (map[string]any) y una
// función para reconstruir el valor de salida respetando si la entrada era un
// recurso único o una lista. Los valores que no son recursos se conservan tal
// cual.
func comoRecursos(data any) ([]map[string]any, func([]map[string]any) any) {
	switch v := data.(type) {
	case map[string]any:
		return []map[string]any{v}, func(rs []map[string]any) any {
			if len(rs) == 0 {
				return nil
			}
			return rs[0]
		}
	case []map[string]any:
		return v, func(rs []map[string]any) any {
			lista := make([]any, len(rs))
			for i := range rs {
				lista[i] = rs[i]
			}
			return lista
		}
	case []any:
		recursos := make([]map[string]any, 0, len(v))
		indices := make([]int, 0, len(v))
		for i := range v {
			if m, ok := v[i].(map[string]any); ok {
				recursos = append(recursos, m)
				indices = append(indices, i)
			}
		}
		return recursos, func(rs []map[string]any) any {
			for k, idx := range indices {
				v[idx] = rs[k]
			}
			return v
		}
	default:
		return nil, func([]map[string]any) any { return data }
	}
}

// expandirValor expande un recurso o una lista de recursos a un nivel dado,
// agrupando las consultas de cada relación entre todos los recursos del nivel
// para evitar el problema N+1.
func (ex *expansor) expandirValor(
	data any,
	relaciones []config.RelationConfig,
	seleccion *NodoSeleccion,
	profundidadRestante int,
	rutaRama []string,
) (any, error) {
	recursos, reconstruir := comoRecursos(data)
	if len(recursos) == 0 {
		return reconstruir(recursos), nil
	}
	if err := ex.expandirNivel(recursos, relaciones, seleccion, profundidadRestante, rutaRama); err != nil {
		return nil, err
	}
	return reconstruir(recursos), nil
}

// expandirNivel expande cada relación aplicable sobre todos los recursos del
// nivel, agrupando las llaves por relación.
func (ex *expansor) expandirNivel(
	recursos []map[string]any,
	relaciones []config.RelationConfig,
	seleccion *NodoSeleccion,
	profundidadRestante int,
	rutaRama []string,
) error {
	if profundidadRestante <= 0 {
		return nil
	}

	for i := range relaciones {
		rel := relaciones[i]

		// Si hay selección, solo expandir las relaciones seleccionadas.
		var subSeleccion *NodoSeleccion
		if seleccion != nil {
			hijo, ok := seleccion.Hijo(rel.MappedName)
			if !ok {
				continue
			}
			subSeleccion = hijo
		}

		if err := ex.expandirRelacionNivel(
			recursos, rel, subSeleccion, profundidadRestante, rutaRama,
		); err != nil {
			return err
		}
	}

	return nil
}

// expandirRelacionNivel resuelve una relación para todos los recursos del nivel
// con una consulta agrupada (dedupe + lotes), aplica ciclos y profundidad
// efectiva, distribuye por related_field, recorre descendientes y aplica la
// política de error.
func (ex *expansor) expandirRelacionNivel(
	recursos []map[string]any,
	rel config.RelationConfig,
	subSeleccion *NodoSeleccion,
	profundidadRestante int,
	rutaRama []string,
) error {
	rutaActual := append(append([]string{}, rutaRama...), rel.MappedName)
	rutaStr := strings.Join(rutaActual, ".")

	// Detección de ciclos: si el mapped_name ya está en la rama, detener.
	for _, visto := range rutaRama {
		if visto == rel.MappedName {
			ex.agregarAviso(responses.RelationError{
				Path:    rutaStr,
				Service: rel.RelatedService,
				Message: "rama detenida por ciclo de relaciones detectado",
				Kind:    KindCycle,
			})
			return nil
		}
	}

	// Profundidad efectiva = min(global restante, depth de la relación).
	efectiva := profundidadRestante
	if rel.Depth < efectiva {
		efectiva = rel.Depth
		ex.agregarAviso(responses.RelationError{
			Path:    rutaStr,
			Service: rel.RelatedService,
			Message: fmt.Sprintf("profundidad reducida a %d por configuración de la relación", rel.Depth),
			Kind:    KindDepth,
		})
	}
	if efectiva <= 0 {
		return nil
	}

	// Recolectar la llave local de cada recurso.
	llaves := make([]string, 0, len(recursos))
	for _, recurso := range recursos {
		if llave, ok := ex.llaveLocal(recurso, rel); ok {
			llaves = append(llaves, llave)
		}
	}

	// Consulta agrupada para todas las llaves del nivel (dedupe + lotes + caché).
	registros, generico, err := ex.resolver(rel, llaves, rutaStr)
	if err != nil {
		return ex.manejarErrorRelacionNivel(recursos, rel, rutaStr)
	}
	if generico && len(llaves) > 0 {
		ex.agregarAviso(AvisoGenericResolver(rutaStr, rel))
	}

	// Una relación marcada Completo (nombrada como hoja en fields) se expande por
	// configuración, como si no hubiera selección por debajo de ella: se ignoran
	// los Campos/Hijos acumulados por otras rutas y no se recorta. Un subSeleccion
	// nil (sin fields) tiene el mismo efecto. Para un nodo parcial, seleccionEfectiva
	// es el propio nodo y gobierna la recursión y la proyección.
	seleccionEfectiva := subSeleccion
	if subSeleccion != nil && subSeleccion.Completo {
		seleccionEfectiva = nil
	}

	// Recursión sobre descendientes de todos los registros resueltos del nivel.
	if len(rel.NestedRelations) > 0 && efectiva-1 > 0 && len(registros) > 0 {
		if err := ex.expandirNivel(
			registros, rel.NestedRelations, seleccionEfectiva, efectiva-1, rutaActual,
		); err != nil {
			return err
		}
	}

	// Distribuir las coincidencias a cada recurso por su llave local.
	for _, recurso := range recursos {
		llave, ok := ex.llaveLocal(recurso, rel)
		if !ok {
			ex.insertarVacio(recurso, rel)
			continue
		}
		coincidencias := filtrarPorRelatedField(registros, rel, llave)
		ex.insertarResultado(recurso, rel, coincidencias)
	}

	// Proyección estricta: cuando hay selección parcial de fields para este nivel,
	// cada registro resuelto se recorta a los escalares seleccionados más las
	// relaciones hijas materializadas. Un nodo Completo (seleccionEfectiva nil) no
	// se recorta. Se aplica después de distribuir (la llave de join ya cumplió su
	// papel) y sobre los mismos punteros insertados.
	proyectarRegistros(registros, seleccionEfectiva)

	// Recorte por lista negra: sobre el mismo choke-point y después de la
	// proyección, se eliminan los campos vetados de cada registro resuelto, sea
	// cual sea el resolver (específico o genérico) y exista o no selección. La
	// lista negra gana sobre allowed_fields y sobre el Id implícito (precedencia
	// lista negra > allowed_fields > Id), y su supresión es silenciosa (sin aviso).
	ex.recortarVetados(registros, rel)

	return nil
}

// recortarVetados elimina in situ de cada registro resuelto los campos vetados
// por la lista negra para la relación (emparejando por RelatedAPI/RelatedTable,
// con los comodines WildcardANY). Se aplica sobre la salida de todo resolver,
// después de la proyección por fields, de modo que un campo permitido por
// allowed_fields o el Id implícito también se suprimen si están vetados. Una
// lista negra nil o vacía no recorta nada.
func (ex *expansor) recortarVetados(registros []map[string]any, rel config.RelationConfig) {
	if len(ex.blacklist) == 0 {
		return
	}
	for _, reg := range registros {
		for campo := range reg {
			if ex.blacklist.CampoVetado(rel.RelatedAPI, rel.RelatedTable, campo) {
				delete(reg, campo)
			}
		}
	}
}

// proyectarRegistros recorta in situ cada registro a los campos seleccionados
// cuando la selección del nivel no está vacía (contiene campos escalares o
// relaciones hijas). Conserva los campos escalares en seleccion.Campos y los
// mapped_name de las relaciones hijas seleccionadas; elimina cualquier otro
// campo escalar, incluido el Id cuando no se seleccionó. Cuando la selección es
// nil (relación pedida completa) o vacía, no recorta: devuelve el registro tal
// como lo entregó el resolver.
func proyectarRegistros(registros []map[string]any, seleccion *NodoSeleccion) {
	if seleccion == nil || seleccion.Vacio() {
		return
	}
	for _, reg := range registros {
		for campo := range reg {
			if _, esEscalarSeleccionado := seleccion.Campos[campo]; esEscalarSeleccionado {
				continue
			}
			if _, esRelacionHija := seleccion.Hijos[campo]; esRelacionHija {
				continue
			}
			delete(reg, campo)
		}
	}
}

// llaveLocal obtiene la llave local del recurso según la dirección de la
// relación. Devuelve la llave como string y si está presente y no vacía.
func (ex *expansor) llaveLocal(recurso map[string]any, rel config.RelationConfig) (string, bool) {
	valor, existe := recurso[rel.LocalField]
	if !existe || valor == nil {
		return "", false
	}
	llave := aCadena(valor)
	if strings.TrimSpace(llave) == "" {
		return "", false
	}
	return llave, true
}

// resolver consulta el registro apropiado, usa caché por solicitud y respeta
// max_related_ids. Devuelve los registros, si el resolver fue genérico y error.
func (ex *expansor) resolver(
	rel config.RelationConfig,
	llaves []string,
	rutaStr string,
) ([]map[string]any, bool, error) {
	res, generico := ex.registry.resolverPara(rel)
	if res == nil {
		return nil, false, fmt.Errorf("expand: no hay resolver para la relación %q", rutaStr)
	}

	// Deduplicar llaves y consultar solo las no cacheadas.
	pendientes := make([]string, 0, len(llaves))
	acumulado := make([]map[string]any, 0)
	for _, k := range dedupe(llaves) {
		ck := ex.claveCache(rel, k)
		if cacheado, ok := ex.cache[ck]; ok {
			acumulado = append(acumulado, cacheado...)
			continue
		}
		pendientes = append(pendientes, k)
	}

	for _, lote := range particionar(pendientes, ex.maxRelIDs) {
		registros, err := res.Resolve(ex.ctx, rel, lote)
		if err != nil {
			return nil, generico, err
		}
		// Guardar en caché por llave según related_field.
		for _, k := range lote {
			ck := ex.claveCache(rel, k)
			ex.cache[ck] = filtrarPorRelatedField(registros, rel, k)
		}
		acumulado = append(acumulado, registros...)
	}

	return acumulado, generico, nil
}

func (ex *expansor) claveCache(rel config.RelationConfig, llave string) string {
	return rel.RelatedService + "|" + rel.MappedName + "|" + rel.RelatedField + "|" + llave
}

// manejarErrorRelacionNivel aplica la política on_relation_error a nivel de todo
// el nivel. En modo fail aborta con *errorFatalExpansion. En modo notify conserva
// cada recurso principal, deja la rama vacía (null/[]) y agrega un único aviso
// relation no sensible.
func (ex *expansor) manejarErrorRelacionNivel(
	recursos []map[string]any,
	rel config.RelationConfig,
	rutaStr string,
) error {
	if ex.onRelError == config.OnRelationErrorFail {
		return &errorFatalExpansion{path: rutaStr, servicio: rel.RelatedService}
	}
	for _, recurso := range recursos {
		ex.insertarVacio(recurso, rel)
	}
	ex.agregarAviso(responses.RelationError{
		Path:    rutaStr,
		Service: rel.RelatedService,
		Message: "no se pudo resolver la relación; se conserva el recurso principal",
		Kind:    KindRelation,
	})
	return nil
}

// insertarResultado inserta las coincidencias bajo mapped_name respetando la
// cardinalidad y conservando la llave local.
func (ex *expansor) insertarResultado(
	recurso map[string]any,
	rel config.RelationConfig,
	coincidencias []map[string]any,
) {
	if rel.Cardinality == config.CardinalityMany {
		lista := make([]any, len(coincidencias))
		for i := range coincidencias {
			lista[i] = coincidencias[i]
		}
		recurso[rel.MappedName] = lista
		return
	}
	// one: primer resultado o null.
	if len(coincidencias) == 0 {
		recurso[rel.MappedName] = nil
		return
	}
	recurso[rel.MappedName] = coincidencias[0]
}

// insertarVacio inserta el valor vacío de la cardinalidad: null para one, [] para
// many.
func (ex *expansor) insertarVacio(recurso map[string]any, rel config.RelationConfig) {
	if rel.Cardinality == config.CardinalityMany {
		recurso[rel.MappedName] = []any{}
		return
	}
	recurso[rel.MappedName] = nil
}

// agregarAviso añade un aviso a la colección pública si no está duplicado por
// path+kind.
func (ex *expansor) agregarAviso(aviso responses.RelationError) {
	for _, a := range ex.warnings {
		if a.Path == aviso.Path && a.Kind == aviso.Kind {
			return
		}
	}
	ex.warnings = append(ex.warnings, aviso)
}

// filtrarPorRelatedField devuelve los registros cuyo related_field coincide con
// la llave local indicada.
func filtrarPorRelatedField(registros []map[string]any, rel config.RelationConfig, llave string) []map[string]any {
	salida := make([]map[string]any, 0, len(registros))
	for _, reg := range registros {
		valor, ok := reg[rel.RelatedField]
		if !ok {
			continue
		}
		if aCadena(valor) == llave {
			salida = append(salida, reg)
		}
	}
	return salida
}

// dedupe elimina duplicados conservando el orden de aparición.
func dedupe(llaves []string) []string {
	vistos := make(map[string]struct{}, len(llaves))
	salida := make([]string, 0, len(llaves))
	for _, k := range llaves {
		if _, ok := vistos[k]; ok {
			continue
		}
		vistos[k] = struct{}{}
		salida = append(salida, k)
	}
	return salida
}

// particionar divide las llaves en lotes de a lo sumo tamano elementos. Un tamano
// <= 0 devuelve un único lote con todas las llaves.
func particionar(llaves []string, tamano int) [][]string {
	if len(llaves) == 0 {
		return nil
	}
	if tamano <= 0 {
		return [][]string{llaves}
	}
	var lotes [][]string
	for i := 0; i < len(llaves); i += tamano {
		fin := i + tamano
		if fin > len(llaves) {
			fin = len(llaves)
		}
		lotes = append(lotes, llaves[i:fin])
	}
	return lotes
}

// aCadena normaliza un valor de llave a string para comparaciones estables entre
// números decodificados de JSON (float64) y cadenas.
func aCadena(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		// Evitar notación científica para enteros.
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%v", t)
	case int:
		return fmt.Sprintf("%d", t)
	case int64:
		return fmt.Sprintf("%d", t)
	case bool:
		return fmt.Sprintf("%v", t)
	default:
		return fmt.Sprintf("%v", t)
	}
}
