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
// registro materializado, que debe incluir su identificador. Un payload con Id
// identifica el registro existente y se actualiza; sin Id se crea. La existencia
// la decide el motor en el pre-pass con Recuperar, por lo que Materialize no vuelve
// a consultar el CRUD. keys es la clave natural declarada, si la hay, y se pasa
// solo como información.
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

	// Recuperar devuelve, usando únicamente lecturas (GET), el registro existente
	// de la entidad para el payload dado: por el Id presente en el payload o por la
	// clave natural declarada (keys). Devuelve nil, sin error, cuando no existe un
	// registro reutilizable, y nunca ejecuta una escritura.
	//
	// El motor la invoca en el pre-pass de solo lectura, antes de la primera
	// escritura, y retiene el registro para decidir los conflictos de llave.
	Recuperar(
		ctx context.Context,
		rel config.RelationConfig,
		payload map[string]any,
		keys []string,
	) (map[string]any, error)

	// Eliminar da de baja, en el CRUD dueño, el registro con el Id dado. El motor la
	// invoca solo para un nodo marcado con _Eliminar cuya arista admite deletable, y
	// nunca por inferencia a partir de nodos ausentes.
	Eliminar(ctx context.Context, rel config.RelationConfig, id any) error
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

// Fases de eventoEscritura que el motor reporta al observador interno de las pruebas.
const (
	// faseLectura indica la recuperación por lectura (GET) del registro de un nodo.
	faseLectura = "lectura"
	// faseDecision indica que los controles de un nodo recuperado se validaron con éxito.
	faseDecision = "decision"
	// faseEscritura indica una escritura o baja enviada a un CRUD dueño.
	faseEscritura = "escritura"
)

// eventoEscritura describe una acción del motor de escritura para la observación interna.
type eventoEscritura struct {
	// Fase es faseLectura, faseDecision o faseEscritura.
	Fase string
	// Ruta es la ruta del nodo en el cuerpo, con la misma notación que los errores.
	Ruta string
	// Servicio es el servicio dueño del nodo. Vacío en la raíz se resuelve al servicio raíz.
	Servicio string
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
	// servicioRaiz es el servicio de la entidad raíz, para identificar sus fallos.
	servicioRaiz string
	// retenido guarda, por identidad de nodo, el estado recuperado en el pre-pass.
	// Vive solo durante una invocación de Write y se descarta al terminar.
	retenido map[string]nodoRetenido
	// observador recibe los eventos de fase de writeObservado; nil no emite.
	observador func(eventoEscritura)
}

// observar emite un evento de fase al observador, si lo hay. Un servicio vacío se
// resuelve al servicio de la entidad raíz.
func (me *motorEscritura) observar(fase, ruta, servicio string) {
	if me.observador == nil {
		return
	}
	if servicio == "" {
		servicio = me.servicioRaiz
	}
	me.observador(eventoEscritura{Fase: fase, Ruta: ruta, Servicio: servicio})
}

// Write materializa un cuerpo anidado en cascada a partir de la configuración de
// escritura del servicio. Recorre el árbol en orden topológico (padre antes que
// hijos), propaga la llave del padre a cada hijo según el modo declarado, aplana
// los parámetros anidados a llave foránea y delega la materialización de cada
// entidad en su WriteResolver.
//
// Validación previa sin materializar: antes de cualquier escritura, el cuerpo
// completo se valida y aplana contra las políticas de escritura declaradas; un
// campo no escribible devuelve *ErrorEscritura. Después, el pre-pass de solo lectura
// recupera el estado de todo el árbol: un campo inmutable que difiere del registro
// existente o una violación de política o de no-reparent devuelve *ErrorEscritura
// (que el llamador mapea a 400) sin materializar nada.
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
	return writeObservado(ctx, body, cfg, registry, options, nil)
}

// writeObservado implementa Write y reporta al observador, si no es nil, cada lectura,
// decisión validada y escritura en el orden en que ocurren. Solo lo usan las pruebas del
// paquete para verificar que la validación precede a las escrituras.
func writeObservado(
	ctx context.Context,
	body map[string]any,
	cfg config.ServiceConfig,
	registry WriteResolverRegistry,
	options WriteOptions,
	observador func(eventoEscritura),
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

	body = desaplanarEspecializaciones(body, *cfg.Write, cfg.Relations)

	// Validación y aplanado de todo el árbol antes de materializar. Un cuerpo
	// inválido se rechaza con *ErrorEscritura sin ninguna escritura.
	if err := validarNivel(body, *cfg.Write, cfg.Relations, verb, "", nil); err != nil {
		return WriteResult{}, err
	}

	me := &motorEscritura{
		ctx:          ctx,
		registry:     registry,
		verb:         verb,
		onRelError:   onRelError,
		servicioRaiz: cfg.Service,
		retenido:     make(map[string]nodoRetenido),
		observador:   observador,
	}

	// Pre-pass de solo lectura sobre todo el árbol: recupera y retiene el estado de
	// cada nodo y decide los conflictos de llave antes de la primera escritura. Un
	// 400 aquí no deja escrituras parciales en ningún CRUD dueño.
	if err := me.prepass(body, registry.resolverRaiz(), config.RelationConfig{}, *cfg.Write, cfg.Relations, "", nil); err != nil {
		return WriteResult{}, err
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
// materializar. Rechaza con *ErrorEscritura un campo no escribible y una relación no
// declarada escribible. La comparación de inmutables contra el registro existente es
// del pre-pass (validarInmutables), no de este recorrido de forma.
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

		// La marca _Eliminar es reservada: debe ser booleana y no puede aplicarse a la
		// raíz, que no es una arista con política de eliminación. No es un campo del
		// objeto, así que no se valida como escribible ni se envía al CRUD.
		if clave == campoEliminar {
			marca, esBool := valor.(bool)
			if !esBool {
				return &ErrorEscritura{Path: rutaCampo, Motivo: motivoMarcaEliminarNoBooleana}
			}
			if marca && ruta == "" {
				return &ErrorEscritura{Path: rutaCampo, Motivo: motivoMarcaEliminarEnRaiz}
			}
			continue
		}

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
		if rel.Write.KeyPropagation == config.KeyPropagationFKInbound && esMarcaEliminar(v) {
			return &ErrorEscritura{Path: ruta, Motivo: motivoEliminarFKInbound}
		}
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

// motivoConflictoLlave describe el conflicto de una llave reenviada que difiere de
// la autoritativa en una entidad existente. No expone valores del cuerpo ni del CRUD.
const motivoConflictoLlave = "la llave relacional reenviada difiere de la autoritativa en una " +
	"entidad existente; no se reasigna de forma implícita"

// nodoRetenido es el estado de un nodo del cuerpo que el pre-pass conserva: el
// registro existente recuperado por lectura o la marca de omisión cuando la lectura
// falló y la política de error tolerada descarta el nodo y su subárbol.
type nodoRetenido struct {
	registro map[string]any
	omitido  bool
	// accion es la acción clasificada para el nodo a partir de registro.
	accion AccionEscritura
}

// AccionEscritura es la acción que el motor materializará para un nodo. La decide el
// pre-pass a partir del registro existente recuperado, antes de cualquier escritura.
type AccionEscritura string

const (
	// AccionCrear indica que el nodo no resuelve a un registro existente.
	AccionCrear AccionEscritura = "crear"
	// AccionActualizar indica que el nodo resuelve a un registro existente y reenvía
	// al menos un campo escribible distinto del almacenado.
	AccionActualizar AccionEscritura = "actualizar"
	// AccionSinCambios indica que el nodo resuelve a un registro existente y sus campos
	// escribibles coinciden con los almacenados. Es una reconciliación, no una
	// actualización.
	AccionSinCambios AccionEscritura = "sin_cambios"
	// AccionEliminar indica que el nodo está marcado con _Eliminar y su registro existente
	// se da de baja. No se materializa como creación ni actualización.
	AccionEliminar AccionEscritura = "eliminar"
)

// clasificarAccion decide la acción de un nodo. Sin registro existente, o con un
// registro sin Id, la acción es crear. Con registro existente, compara solo los campos
// escribibles reenviados que no son derivados: si alguno difiere del almacenado la
// acción es actualizar; si todos coinciden es sin cambios. Un registro que solo trae
// Id no permite confirmar que los campos coinciden, así que cualquier campo reenviado
// cuenta como cambio.
func clasificarAccion(
	plano map[string]any,
	registro map[string]any,
	escribibles []string,
	derivados map[string]struct{},
) AccionEscritura {
	if registro == nil || registro["Id"] == nil {
		return AccionCrear
	}
	esc := conjunto(escribibles)
	for campo, valor := range plano {
		if campo == "Id" || !contiene(esc, campo) || contiene(derivados, campo) {
			continue
		}
		if cadenaONula(valor) != cadenaONula(registro[campo]) {
			return AccionActualizar
		}
	}
	return AccionSinCambios
}

// Motivos de rechazo de la política de escritura por relación. No exponen valores
// del cuerpo ni del CRUD.
const (
	motivoCreacionNoPermitida      = "la política de la relación no permite crear esta entidad (creatable: false)"
	motivoActualizacionNoPermitida = "la política de la relación no permite actualizar esta entidad (updatable: false)"
)

// campoEliminar es la marca reservada que el servicio llamante pone sobre un objeto
// lógico para pedir su baja. No pertenece al objeto: el motor la valida como booleana
// y la retira de todo lo que envía a los CRUD.
const campoEliminar = "_Eliminar"

// Motivos de rechazo de la marca _Eliminar. No exponen valores del cuerpo ni del CRUD.
const (
	motivoMarcaEliminarNoBooleana       = "la marca _Eliminar debe ser booleana"
	motivoMarcaEliminarEnRaiz           = "la raíz no admite eliminación por la marca _Eliminar"
	motivoEliminacionNoPermitida        = "la política de la relación no permite eliminar esta entidad (deletable: false)"
	motivoEliminarSinIdConClaveDerivada = "sin Id, la clave natural incluye un campo derivado de la relación; no se puede identificar el registro a eliminar"
	motivoEliminarFKInbound             = "la eliminación por marca no aplica a una entidad referenciada por FK ascendente (fk_inbound)"
)

// esMarcaEliminar informa si el objeto lógico lleva la marca _Eliminar en true.
func esMarcaEliminar(nodo map[string]any) bool {
	marca, ok := nodo[campoEliminar].(bool)
	return ok && marca
}

// validarPoliticaAccion aplica la política de verbos de la arista a la acción que el
// pre-pass clasificó para el nodo. Crear exige creatable y actualizar exige updatable.
// Una acción sin cambios no materializa ninguna escritura y no se gatea, de modo que
// el reenvío de una entidad existente sin cambios no se rechaza.
//
// ruta identifica la arista en el error. Devuelve *ErrorEscritura si la acción no está
// permitida por la política, y nil en otro caso.
func validarPoliticaAccion(accion AccionEscritura, pol config.WriteVerbPolicy, ruta string) error {
	switch accion {
	case AccionCrear:
		if !pol.Creatable {
			return &ErrorEscritura{Path: ruta, Motivo: motivoCreacionNoPermitida}
		}
	case AccionActualizar:
		if !pol.Updatable {
			return &ErrorEscritura{Path: ruta, Motivo: motivoActualizacionNoPermitida}
		}
	}
	return nil
}

// cadenaONula normaliza un valor para comparar campos: nil equivale a la cadena vacía.
func cadenaONula(v any) string {
	if v == nil {
		return ""
	}
	return aCadena(v)
}

// claveNodo identifica un nodo del cuerpo por la identidad de su mapa, de modo que
// su estado retenido sigue al nodo aunque su ruta lógica se repita en un arreglo.
func claveNodo(nodo map[string]any) string {
	return fmt.Sprintf("%p", nodo)
}

// accionDe devuelve la acción clasificada en el pre-pass para el nodo. Devuelve el
// valor vacío si el nodo no fue recuperado.
func (me *motorEscritura) accionDe(nodo map[string]any) AccionEscritura {
	return me.retenido[claveNodo(nodo)].accion
}

// omitido informa si el pre-pass marcó el nodo como omitido por un fallo de lectura
// tolerado, de modo que la materialización debe descartarlo.
func (me *motorEscritura) omitido(nodo map[string]any) bool {
	return me.retenido[claveNodo(nodo)].omitido
}

// llaveDelNodo devuelve la llave autoritativa conocida de un nodo: su Id reenviado o,
// si no lo trae, el Id de su registro existente. Devuelve nil cuando el nodo aún no
// existe en el CRUD dueño y su Id lo asignará la creación.
func llaveDelNodo(nodo map[string]any, registro map[string]any) any {
	if id := nodo["Id"]; id != nil {
		return id
	}
	if id := registro["Id"]; id != nil {
		return id
	}
	return nil
}

// hayConflictoLlave informa si un nodo con Id propio reenvía el campo derivado con un
// valor distinto de la llave del padre. Con llavePadre nil (el padre se creará con un
// Id nuevo) cualquier valor reenviado es un conflicto. Sin Id propio no hay conflicto:
// la llave se canoniza en la materialización.
func hayConflictoLlave(nodo map[string]any, campo string, llavePadre any) bool {
	reenviado, presente := nodo[campo]
	if !presente || reenviado == nil {
		return false
	}
	if id, hay := nodo["Id"]; !hay || id == nil {
		return false
	}
	if llavePadre == nil {
		return true
	}
	return aCadena(reenviado) != aCadena(llavePadre)
}

// reasociaExistente informa si un registro existente tiene asociado el campo derivado a
// un padre distinto de llavePadre. Un valor almacenado nulo no es una asociación y no
// cuenta. Con llavePadre nil (el padre aún no existe) cualquier asociación almacenada
// implica reasignar el hijo a otro padre.
//
// Solo aplica a llaves foráneas. En una propagación por PK compartida el campo derivado
// es la propia identidad del registro; su conflicto lo cubre hayConflictoLlave.
func reasociaExistente(registro map[string]any, campo string, llavePadre any) bool {
	if campo == "Id" || registro == nil || registro["Id"] == nil {
		return false
	}
	almacenado := registro[campo]
	if almacenado == nil {
		return false
	}
	if llavePadre == nil {
		return true
	}
	return aCadena(almacenado) != aCadena(llavePadre)
}

// prepass recorre en solo lectura el nodo y su subárbol, recupera el registro
// existente de cada uno y lo retiene en me.retenido. Decide los conflictos de llave
// antes de cualquier escritura. Un nodo cuya lectura falla se marca como omitido
// según la política de error de relación, y su subárbol no se materializa.
//
// rel es la relación que produjo el nodo (vacía en la raíz) y llavePadre la llave
// autoritativa del padre, o nil si el padre aún no existe en el CRUD dueño.
// Devuelve *ErrorEscritura ante una llave incompatible y *errorFatalEscritura ante
// un fallo de lectura de la raíz o un fallo bajo política estricta.
func (me *motorEscritura) prepass(
	nodo map[string]any,
	res WriteResolver,
	rel config.RelationConfig,
	wc config.WriteConfig,
	relNivel []config.RelationConfig,
	ruta string,
	llavePadre any,
) error {
	esRaiz := rel.RelatedService == ""
	if res == nil {
		return fmt.Errorf("write: no hay resolver para %q", ruta)
	}

	plano := aplanarParametros(nodo, wc.WritableFields, relNivel, camposDerivadosInternos(rel, relNivel))
	keys := valoresClaveNatural(plano, wc.NaturalKey)
	registro, err := res.Recuperar(me.ctx, rel, plano, keys)
	me.observar(faseLectura, ruta, rel.RelatedService)
	if err != nil {
		if esRaiz {
			return &errorFatalEscritura{path: "(raíz)", servicio: me.servicioRaiz, err: err}
		}
		if err := me.manejarErrorRelacion(rel, ruta); err != nil {
			return err
		}
		me.retenido[claveNodo(nodo)] = nodoRetenido{omitido: true}
		return nil
	}
	if err := validarInmutables(plano, registro, wc, camposDerivadosInternos(rel, relNivel), me.verb, ruta); err != nil {
		return err
	}
	if esMarcaEliminar(nodo) {
		if err := me.prepassEliminar(nodo, rel, wc, relNivel, ruta, registro); err != nil {
			return err
		}
		me.observar(faseDecision, ruta, rel.RelatedService)
		return nil
	}

	accion := clasificarAccion(plano, registro, wc.WritableFields, camposDerivadosInternos(rel, relNivel))
	me.retenido[claveNodo(nodo)] = nodoRetenido{registro: registro, accion: accion}

	// La política de la arista gatea la acción materializada. La raíz no es una
	// arista y no tiene política por relación.
	if !esRaiz {
		if err := validarPoliticaAccion(accion, wc.Policy, ruta); err != nil {
			return err
		}
	}

	if !esRaiz && rel.Write != nil && rel.Write.KeyPropagation != config.KeyPropagationFKInbound {
		if campo, ok := rel.CampoDerivado(); ok &&
			(hayConflictoLlave(nodo, campo, llavePadre) || reasociaExistente(registro, campo, llavePadre)) {
			return &ErrorEscritura{Path: ruta, Motivo: motivoConflictoLlave}
		}
	}

	me.observar(faseDecision, ruta, rel.RelatedService)
	return me.prepassHijos(nodo, relNivel, esRaiz, ruta, llaveDelNodo(nodo, registro))
}

// prepassEliminar decide la baja de un nodo marcado con _Eliminar. Exige deletable en
// la arista y, sin Id, rechaza una clave natural que incluya un campo derivado de la
// relación, porque el registro no podría identificarse. Si el nodo no tiene registro
// que borrar, se omite con un aviso no bloqueante. El subárbol de un nodo marcado no
// se recorre: la baja no escribe sus hijos ni los infiere.
//
// Devuelve *ErrorEscritura por una marca no permitida o no identificable. El registro
// retenido, cuando existe, lo usa materializarValorRelacion para emitir el DELETE.
func (me *motorEscritura) prepassEliminar(
	nodo map[string]any,
	rel config.RelationConfig,
	wc config.WriteConfig,
	relNivel []config.RelationConfig,
	ruta string,
	registro map[string]any,
) error {
	if !wc.Policy.Deletable {
		return &ErrorEscritura{Path: ruta, Motivo: motivoEliminacionNoPermitida}
	}
	if nodo["Id"] == nil && claveIncluyeDerivado(wc.NaturalKey, camposDerivadosInternos(rel, relNivel)) {
		return &ErrorEscritura{Path: ruta, Motivo: motivoEliminarSinIdConClaveDerivada}
	}
	if registro == nil || registro["Id"] == nil {
		me.retenido[claveNodo(nodo)] = nodoRetenido{omitido: true}
		me.agregarAviso(responses.RelationError{
			Path:    ruta,
			Service: rel.RelatedService,
			Message: "no hay registro que eliminar; la baja se omite",
			Kind:    KindEliminacionOmitida,
		})
		return nil
	}
	me.retenido[claveNodo(nodo)] = nodoRetenido{registro: registro, accion: AccionEliminar}
	return nil
}

// validarInmutables rechaza, en actualización, un campo inmutable reenviado cuyo valor
// difiere del registro existente. Un valor igual al almacenado no es una modificación.
// Sin registro existente no hay valor previo que modificar, y los campos derivados se
// conciliaron aparte, así que ambos casos se omiten.
//
// ruta identifica el nodo; devuelve *ErrorEscritura con el campo responsable.
func validarInmutables(
	plano map[string]any,
	registro map[string]any,
	wc config.WriteConfig,
	derivados map[string]struct{},
	verb string,
	ruta string,
) error {
	if verb != WriteVerbUpdate || registro == nil || registro["Id"] == nil {
		return nil
	}
	for _, campo := range wc.ImmutableFields {
		if contiene(derivados, campo) {
			continue
		}
		valor, presente := plano[campo]
		if !presente {
			continue
		}
		if cadenaONula(valor) != cadenaONula(registro[campo]) {
			return &ErrorEscritura{Path: unirRuta(ruta, campo), Motivo: "campo inmutable; no puede modificarse"}
		}
	}
	return nil
}

// claveIncluyeDerivado informa si la clave natural incluye algún campo derivado de la
// relación, cuyo valor solo se conoce al materializar el padre.
func claveIncluyeDerivado(claveNatural []string, derivados map[string]struct{}) bool {
	for _, campo := range claveNatural {
		if contiene(derivados, campo) {
			return true
		}
	}
	return false
}

// prepassHijos aplica el pre-pass a las relaciones hijas de un nodo ya recuperado.
// Las relaciones fk_inbound se recorren solo desde la raíz, igual que en la
// materialización; el resto desciende con la llave del nodo como llave del padre.
// Para fk_inbound concilia la FK local del nodo con el Id autoritativo recuperado,
// antes de cualquier escritura.
func (me *motorEscritura) prepassHijos(
	nodo map[string]any,
	relaciones []config.RelationConfig,
	esRaiz bool,
	ruta string,
	llaveNodo any,
) error {
	for i := range relaciones {
		hijo := relaciones[i]
		if hijo.Write == nil {
			continue
		}
		fkInbound := hijo.Write.KeyPropagation == config.KeyPropagationFKInbound
		if fkInbound && !esRaiz {
			continue
		}
		valor, presente := nodo[hijo.MappedName]
		if !presente || valor == nil {
			continue
		}

		rutaHijo := unirRuta(ruta, hijo.MappedName)
		res, ok := me.registry.resolverPara(hijo)
		if !ok {
			if err := me.manejarErrorRelacion(hijo, rutaHijo); err != nil {
				return err
			}
			continue
		}

		if fkInbound {
			referida, ok := valor.(map[string]any)
			if !ok {
				return &ErrorEscritura{Path: rutaHijo, Motivo: "una relación fk_inbound debe ser un objeto único"}
			}
			if err := me.prepass(referida, res, hijo, *hijo.Write, hijo.NestedRelations, rutaHijo, nil); err != nil {
				return err
			}
			if me.omitido(referida) {
				continue
			}
			if registro := me.retenido[claveNodo(referida)].registro; registro != nil {
				if err := me.conciliarCampoDerivado(nodo, hijo.LocalField, registro["Id"], hijo.RelatedService, rutaHijo); err != nil {
					return err
				}
			}
			continue
		}

		switch v := valor.(type) {
		case map[string]any:
			if err := me.prepass(v, res, hijo, *hijo.Write, hijo.NestedRelations, rutaHijo, llaveNodo); err != nil {
				return err
			}
		case []any:
			for j := range v {
				elem, ok := v[j].(map[string]any)
				if !ok {
					continue
				}
				if err := me.prepass(elem, res, hijo, *hijo.Write, hijo.NestedRelations, fmt.Sprintf("%s[%d]", rutaHijo, j), llaveNodo); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// materializarInbound materializa las entidades referenciadas por fk_inbound antes
// que la raíz, y propaga su Id hacia la FK local de la raíz. La recuperación de cada
// nodo y la conciliación previa de la FK local ocurren en el pre-pass (prepass), de
// modo que aquí solo se escribe. Las entidades omitidas por un fallo de lectura
// tolerado se descartan.
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

		if me.omitido(referida) {
			continue
		}

		res, ok := me.registry.resolverPara(rel)
		if !ok {
			if err := me.manejarErrorRelacion(rel, rutaStr); err != nil {
				return err
			}
			continue
		}

		// Materializar la entidad referenciada y propagar su Id.
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

// eliminarEntidad da de baja el registro retenido por el pre-pass para un nodo marcado,
// mediante el método de eliminación del WriteResolver de su relación. Un fallo se trata
// con la política de error de relación, igual que una materialización fallida.
func (me *motorEscritura) eliminarEntidad(
	res WriteResolver,
	rel config.RelationConfig,
	elem map[string]any,
	ruta string,
) error {
	id := me.retenido[claveNodo(elem)].registro["Id"]
	me.observar(faseEscritura, ruta, rel.RelatedService)
	if err := res.Eliminar(me.ctx, rel, id); err != nil {
		return me.manejarErrorRelacion(rel, ruta)
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
	padreID any,
	rutaActual []string,
	rutaStr string,
) error {
	procesar := func(elem map[string]any, ruta string) error {
		if me.omitido(elem) {
			return nil
		}
		// Un nodo marcado para baja se da de baja por el método de eliminación y su
		// subárbol no se materializa.
		if me.accionDe(elem) == AccionEliminar {
			return me.eliminarEntidad(res, rel, elem, ruta)
		}
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
	// Un nodo sin cambios no se escribe: el árbol conserva su registro existente. Sus
	// hijos se siguen materializando por separado.
	if me.accionDe(payload) == AccionSinCambios {
		registro := me.retenido[claveNodo(payload)].registro
		salida := make(map[string]any, len(registro)+len(plano))
		for k, v := range registro {
			salida[k] = v
		}
		for k, v := range plano {
			salida[k] = v
		}
		salida["Id"] = registro["Id"]
		return salida, nil
	}
	// Un registro existente retenido por el pre-pass aporta el Id con el que la
	// materialización actualiza, sin volver a consultar el CRUD.
	if _, tieneID := plano["Id"]; !tieneID {
		if id := me.retenido[claveNodo(payload)].registro["Id"]; id != nil {
			plano["Id"] = id
		}
	}
	keys := valoresClaveNatural(plano, wc.NaturalKey)
	me.observar(faseEscritura, ruta, rel.RelatedService)
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
			Path:   ruta,
			Motivo: motivoConflictoLlave,
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

// desaplanarEspecializaciones devuelve una copia del cuerpo en la que los campos planos
// que pertenecen a una relación aplanada (flatten) se agrupan bajo su mapped_name. Así el
// resto del motor los valida y materializa en el CRUD dueño de la especialización, por PK
// compartida. Un campo que no es escribible en la raíz ni en una especialización
// aplanada se deja donde está y falla como campo ajeno.
func desaplanarEspecializaciones(
	body map[string]any,
	raiz config.WriteConfig,
	rels []config.RelationConfig,
) map[string]any {
	enRaiz := conjunto(raiz.WritableFields)
	salida := make(map[string]any, len(body))
	for clave, valor := range body {
		salida[clave] = valor
	}
	for _, rel := range rels {
		if !rel.Flatten || rel.Write == nil {
			continue
		}
		propios := conjunto(rel.Write.WritableFields)
		grupo := map[string]any{}
		for clave, valor := range salida {
			if _, enRaizCampo := enRaiz[clave]; clave == rel.MappedName || enRaizCampo || !esCampoPropio(propios, clave) {
				continue
			}
			grupo[clave] = valor
			delete(salida, clave)
		}
		if len(grupo) == 0 {
			continue
		}
		anidado, _ := salida[rel.MappedName].(map[string]any)
		combinado := make(map[string]any, len(anidado)+len(grupo))
		for clave, valor := range anidado {
			combinado[clave] = valor
		}
		for clave, valor := range grupo {
			combinado[clave] = valor
		}
		salida[rel.MappedName] = combinado
	}
	return salida
}

// esCampoPropio indica si clave pertenece al conjunto de campos escribibles de la relación.
func esCampoPropio(propios map[string]struct{}, clave string) bool {
	_, ok := propios[clave]
	return ok
}
