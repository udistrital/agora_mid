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

	// Recuperar resuelve, usando únicamente lecturas (GET), el identificador
	// autoritativo de un registro existente para el payload dado: por el Id
	// presente en el payload o por la clave natural declarada (keys). Devuelve 0
	// cuando no existe un registro reutilizable y no ejecuta ninguna escritura.
	//
	// El motor la usa para validar ANTES de materializar (por ejemplo, conciliar
	// la FK local de un nodo contra el Id autoritativo de la entidad referenciada
	// por fk_inbound) sin provocar efectos colaterales: un 400 por conflicto no
	// debe dejar escrituras parciales en ningún CRUD dueño.
	Recuperar(
		ctx context.Context,
		rel config.RelationConfig,
		payload map[string]any,
		keys []string,
	) (int, error)
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
	if err := validarNivel(body, *cfg.Write, cfg.Relations, verb, "", nil); err != nil {
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
		registry.resolverRaiz(), config.RelationConfig{}, body, *cfg.Write, "(raíz)", cfg.Relations,
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
//
// Además de los campos escribibles declarados, reconoce los campos relacionales
// derivados permitidos en este nivel sin exigir que estén en writable_fields:
//   - derivadosInyectados: campos que el padre aporta al validar una relación
//     descendente (fk -> related_field del hijo; shared_pk -> Id, ya aceptado);
//   - derivados propios de este nivel: por cada relación fk_inbound declarada en
//     este nivel, su local_field es un campo derivado del propio nodo.
//
// Un campo derivado con valor null se trata como ausente (el motor lo completará
// con la llave autoritativa) y no se valida como campo escribible.
func validarNivel(
	entidad map[string]any,
	wc config.WriteConfig,
	relaciones []config.RelationConfig,
	verb string,
	ruta string,
	derivadosInyectados map[string]struct{},
) error {
	escribibles := conjunto(wc.WritableFields)
	inmutables := conjunto(wc.ImmutableFields)
	porMapped := indexarPorMapped(relaciones)

	// Campos derivados reconocibles en este nivel: los inyectados por el padre más
	// los propios de las relaciones fk_inbound declaradas aquí (su local_field es
	// una FK ascendente del propio nodo).
	derivados := make(map[string]struct{}, len(derivadosInyectados)+len(relaciones))
	for d := range derivadosInyectados {
		derivados[d] = struct{}{}
	}
	for i := range relaciones {
		rel := relaciones[i]
		if rel.Write != nil && rel.Write.KeyPropagation == config.KeyPropagationFKInbound {
			if campo, ok := rel.CampoDerivado(); ok {
				derivados[campo] = struct{}{}
			}
		}
	}

	for clave, valor := range entidad {
		rutaCampo := unirRuta(ruta, clave)

		// Un campo que corresponde a una relación hija escribible se valida como
		// subárbol, no como campo propio. La relación aporta el campo derivado de
		// sus hijos (fk -> related_field).
		if rel, esRel := porMapped[clave]; esRel {
			if rel.Write == nil {
				return &ErrorEscritura{Path: rutaCampo, Motivo: "la relación no admite escritura"}
			}
			if err := validarRelacionAnidada(valor, rel, verb, rutaCampo); err != nil {
				return err
			}
			continue
		}

		// Un campo derivado relacional reconocido no se rechaza como desconocido;
		// null equivale a ausencia (el motor lo completará con la llave
		// autoritativa). La inmutabilidad no aplica aquí: protege contra el cambio
		// de valor, y reenviar un campo derivado no modifica por sí mismo. La
		// conciliación posterior decide según la matriz —coincide conserva; difiere
		// en entidad existente es conflicto (400); difiere en entidad nueva
		// canoniza—, de modo que un reenvío coincidente de una FK inmutable derivada
		// se acepta.
		if contiene(derivados, clave) {
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

// derivadosDescendentes devuelve el conjunto de campos derivados que una relación
// descendente (fk o shared_pk) aporta a cada entidad hija: la FK del hijo
// (related_field) en fk; el Id en shared_pk (que ya se acepta como identificador,
// pero se incluye por completitud). fk_inbound no inyecta nada hacia abajo: su
// campo derivado pertenece al nodo que la declara, no a la entidad referenciada.
func derivadosDescendentes(rel config.RelationConfig) map[string]struct{} {
	if rel.Write == nil || rel.Write.KeyPropagation == config.KeyPropagationFKInbound {
		return nil
	}
	campo, ok := rel.CampoDerivado()
	if !ok {
		return nil
	}
	return map[string]struct{}{campo: {}}
}

// validarRelacionAnidada valida el valor de una relación hija, que puede ser un
// objeto único (cardinalidad 1-1) o una lista (cardinalidad 1-N). Inyecta en la
// validación del hijo el campo derivado descendente que aporta la relación.
func validarRelacionAnidada(
	valor any,
	rel config.RelationConfig,
	verb string,
	ruta string,
) error {
	wc := *rel.Write
	nested := rel.NestedRelations
	inyectados := derivadosDescendentes(rel)
	switch v := valor.(type) {
	case map[string]any:
		return validarNivel(v, wc, nested, verb, ruta, inyectados)
	case []any:
		for i := range v {
			elem, ok := v[i].(map[string]any)
			if !ok {
				return &ErrorEscritura{
					Path:   fmt.Sprintf("%s[%d]", ruta, i),
					Motivo: "cada elemento de la colección debe ser un objeto",
				}
			}
			if err := validarNivel(elem, wc, nested, verb, fmt.Sprintf("%s[%d]", ruta, i), inyectados); err != nil {
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
	// Se propaga con su tipo nativo (ya normalizado a entero por el resolver), sin
	// coaccionarlo a texto. padreID nil indica que no hay llave que propagar.
	var padreID any
	if v, ok := padre["Id"]; ok && v != nil {
		padreID = v
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
// relación del nodo declarada fk_inbound y presente en el cuerpo, recupera por
// lectura el Id autoritativo de la entidad referenciada, concilia la FK local del
// nodo (rel.LocalField) contra ese Id y, solo si no hay conflicto, materializa la
// entidad referenciada (y su propio subárbol) ANTES que el nodo, propagando su Id
// hacia arriba. La entidad referenciada materializada queda anidada en el nodo
// bajo su mapped_name para la respuesta.
//
// Orden "validar antes de escribir": recuperar -> conciliar -> materializar. Un
// conflicto de FK local en un nodo existente devuelve *ErrorEscritura ANTES de
// cualquier escritura, de modo que un 400 no deja escrituras parciales en ningún
// CRUD dueño. Alinea fk_inbound con los modos descendentes (fk, shared_pk), que
// ya concilian antes de materializar.
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

		// 1) Recuperar por lectura el Id autoritativo de la entidad referenciada,
		// sin escribir. Un fallo de la recuperación se trata con la misma política
		// de error de relación que un fallo de escritura (no una ruta distinta).
		idRecuperado, err := me.recuperarEntidad(res, rel, referida, *rel.Write, rel.NestedRelations)
		if err != nil {
			if err := me.manejarErrorRelacion(rel, rutaStr); err != nil {
				return err
			}
			continue
		}

		// 2) Conciliar hacia arriba contra el Id autoritativo recuperado ANTES de
		// materializar: la FK local del nodo (rel.LocalField) se completa, se
		// conserva o, en un nodo existente con FK incompatible, dispara un conflicto
		// (*ErrorEscritura) sin escribir nada. Cuando la entidad referenciada no
		// existe aún (idRecuperado == 0), la conciliación se difiere al Id que la
		// materialización asigne (caso de creación/canonización de entidad nueva).
		if idRecuperado > 0 {
			if err := me.conciliarCampoDerivado(nodo, rel.LocalField, idRecuperado, rel.RelatedService, rutaStr); err != nil {
				return err
			}
		}

		// 3) Sin conflicto: materializar la entidad referenciada y propagar su Id.
		materializado, err := me.materializarEntidad(res, rel, referida, *rel.Write, rutaStr, rel.NestedRelations)
		if err != nil {
			return me.manejarErrorRelacion(rel, rutaStr)
		}
		for k, v := range materializado {
			referida[k] = v
		}

		// Reconciliar la FK local del nodo con el Id autoritativo final de la
		// entidad referenciada. Para un nodo existente ya validado en el paso 2 es
		// un no-op coincidente; para un nodo nuevo con FK desactualizada canoniza y
		// avisa (field_normalization). El disparador del conflicto es el Id del
		// propio nodo, no el de la entidad referenciada.
		if id, hay := referida["Id"]; hay && id != nil {
			if err := me.conciliarCampoDerivado(nodo, rel.LocalField, id, rel.RelatedService, rutaStr); err != nil {
				return err
			}
		}

		// Materializar el subárbol propio de la entidad referenciada (por ejemplo
		// los datos_identificacion del tercero).
		if err := me.materializarHijos(referida, rel.NestedRelations, rutaActual); err != nil {
			return err
		}
	}
	return nil
}

// recuperarEntidad resuelve, usando únicamente lecturas, el Id autoritativo de
// una entidad para la conciliación previa a la escritura. Aplana el payload y
// extrae los valores de la clave natural igual que materializarEntidad, pero
// delega en la operación de recuperación del resolver (Recuperar), que no ejecuta
// escrituras. Devuelve 0 cuando no existe un registro reutilizable.
func (me *motorEscritura) recuperarEntidad(
	res WriteResolver,
	rel config.RelationConfig,
	payload map[string]any,
	wc config.WriteConfig,
	relNivel []config.RelationConfig,
) (int, error) {
	if res == nil {
		return 0, fmt.Errorf("write: no hay resolver para %q", rel.RelatedService)
	}
	plano := aplanarParametros(payload, wc.WritableFields, relNivel, camposDerivadosInternos(rel, relNivel))
	keys := valoresClaveNatural(plano, wc.NaturalKey)
	return res.Recuperar(me.ctx, rel, plano, keys)
}

// materializarValorRelacion materializa el valor de una relación, que puede ser
// un objeto (1-1) o una lista (1-N), propagando la llave del padre a cada
// elemento antes de materializarlo.
func (me *motorEscritura) materializarValorRelacion(
	valor any,
	rel config.RelationConfig,
	res WriteResolver,
	padreID any,
	rutaActual []string,
	rutaStr string,
) error {
	procesar := func(elem map[string]any, ruta string) error {
		// Conciliar la llave del padre con el campo derivado del hijo ANTES de
		// materializar: un conflicto de entidad existente aborta con *ErrorEscritura
		// sin enviar la operación incompatible.
		if err := me.llaveDescendente(elem, rel, padreID, ruta); err != nil {
			return err
		}
		materializado, err := me.materializarEntidad(res, rel, elem, *rel.Write, ruta, rel.NestedRelations)
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
//
// relNivel son las relaciones declaradas en este nivel (las hijas de la entidad):
// de ellas se derivan los campos fk_inbound (local_field) que pertenecen al
// propio nodo y deben conservarse en el payload aunque no sean escribibles por el
// cliente. rel es la relación que produjo esta entidad (vacía para la raíz): si
// es descendente (fk/shared_pk), su campo derivado también vive en esta entidad y
// debe conservarse.
func (me *motorEscritura) materializarEntidad(
	res WriteResolver,
	rel config.RelationConfig,
	payload map[string]any,
	wc config.WriteConfig,
	ruta string,
	relNivel []config.RelationConfig,
) (map[string]any, error) {
	if res == nil {
		return nil, fmt.Errorf("write: no hay resolver para %q", ruta)
	}
	plano := aplanarParametros(payload, wc.WritableFields, relNivel, camposDerivadosInternos(rel, relNivel))
	keys := valoresClaveNatural(plano, wc.NaturalKey)
	return res.Materialize(me.ctx, rel, plano, keys)
}

// camposDerivadosInternos devuelve el conjunto de campos derivados internos que
// viven en una entidad y deben conservarse en su payload aplanado aunque no sean
// escribibles por el cliente:
//   - el campo derivado descendente de la relación que la produjo (fk ->
//     related_field; shared_pk -> Id), que ya quedó validado o canonizado;
//   - la FK local (local_field) de cada relación fk_inbound declarada en este
//     nivel, que el pre-pass ascendente completó o canonizó.
func camposDerivadosInternos(relProductora config.RelationConfig, relNivel []config.RelationConfig) map[string]struct{} {
	campos := make(map[string]struct{})
	if relProductora.Write != nil && relProductora.Write.KeyPropagation != config.KeyPropagationFKInbound {
		if campo, ok := relProductora.CampoDerivado(); ok {
			campos[campo] = struct{}{}
		}
	}
	for i := range relNivel {
		r := relNivel[i]
		if r.Write != nil && r.Write.KeyPropagation == config.KeyPropagationFKInbound {
			if campo, ok := r.CampoDerivado(); ok {
				campos[campo] = struct{}{}
			}
		}
	}
	return campos
}

// conciliarCampoDerivado concilia el valor reenviado de un campo derivado de una
// relación (campoDerivado, dentro de entidad) contra la llave autoritativa
// calculada por el motor, aplicando esta matriz común a fk, shared_pk y
// fk_inbound:
//
//	ausente o null       -> completar con la llave autoritativa, sin aviso
//	igual                -> conservar, sin aviso
//	diferente + sin Id   -> sobrescribir con la llave autoritativa y registrar un aviso
//	diferente + con Id   -> devolver *ErrorEscritura, sin reasignar la entidad
//
// Un Id propio no nulo en entidad indica una entidad existente, único caso en que
// una diferencia es un conflicto; sin Id la diferencia se canoniza. servicio y
// ruta identifican el aviso y el error sin exponer valores ni detalles del CRUD.
//
// Efectos: muta entidad[campoDerivado] con el valor autoritativo (conservando su
// tipo nativo) y, en una canonización, añade un aviso con Kind
// KindFieldNormalization. Si llaveAutoritativa es nil o campoDerivado es vacío no
// hace nada. Devuelve *ErrorEscritura ante un conflicto de entidad existente y nil
// en cualquier otro caso.
func (me *motorEscritura) conciliarCampoDerivado(
	entidad map[string]any,
	campoDerivado string,
	llaveAutoritativa any,
	servicio string,
	ruta string,
) error {
	if llaveAutoritativa == nil || campoDerivado == "" {
		return nil
	}

	reenviado, presente := entidad[campoDerivado]

	// Ausente o null: completar con la llave autoritativa, sin aviso.
	if !presente || reenviado == nil {
		entidad[campoDerivado] = llaveAutoritativa
		return nil
	}

	// Coincide con la llave autoritativa: conservar, sin aviso.
	if aCadena(reenviado) == aCadena(llaveAutoritativa) {
		entidad[campoDerivado] = llaveAutoritativa
		return nil
	}

	// Difiere: el disparador del conflicto es un Id propio (entidad existente).
	if id, hay := entidad["Id"]; hay && id != nil {
		return &ErrorEscritura{
			Path: ruta,
			Motivo: "la llave relacional reenviada difiere de la autoritativa en una " +
				"entidad existente; no se reasigna de forma implícita",
		}
	}

	// Entidad nueva con llave desactualizada: canonizar y avisar (sin exponer
	// valores ni detalles del CRUD).
	entidad[campoDerivado] = llaveAutoritativa
	me.agregarAviso(responses.RelationError{
		Path:    ruta,
		Service: servicio,
		Message: "se normalizó un campo relacional derivado con la llave autoritativa",
		Kind:    KindFieldNormalization,
	})
	return nil
}

// llaveDescendente concilia la llave del padre (padreID) con el campo derivado
// del hijo para los modos descendentes fk y shared_pk, delegando la matriz de
// comparación en conciliarCampoDerivado. padreID nil, un modo sin campo derivado
// o fk_inbound no producen cambios. Devuelve *ErrorEscritura ante un conflicto de
// entidad existente y nil en otro caso.
func (me *motorEscritura) llaveDescendente(
	hijo map[string]any,
	rel config.RelationConfig,
	padreID any,
	ruta string,
) error {
	if padreID == nil {
		return nil
	}
	campo, ok := rel.CampoDerivado()
	if !ok {
		return nil
	}
	return me.conciliarCampoDerivado(hijo, campo, padreID, rel.RelatedService, ruta)
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
// relaciones hijas (que se materializan por separado). Conserva los campos
// escribibles declarados, el Id, las FK aplanadas y los campos derivados internos
// (derivadosInternos) que el motor validó o generó —por ejemplo la FK propagada
// por el padre— aunque no sean escribibles por el cliente, para que lleguen al
// CRUD dueño con su tipo nativo.
func aplanarParametros(
	payload map[string]any,
	writable []string,
	nested []config.RelationConfig,
	derivadosInternos map[string]struct{},
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
		// Un campo derivado interno validado o generado se conserva con su tipo
		// nativo, aunque no sea escribible por el cliente.
		if contiene(derivadosInternos, clave) {
			salida[clave] = valor
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
