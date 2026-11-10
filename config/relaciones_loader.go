package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

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

// cargarServicios carga y valida todos los archivos config/relaciones_*.yaml del
// directorio indicado, resolviendo related_service contra urls. Un archivo
// ausente, inválido o con una relación inconsistente devuelve un error que
// identifica el archivo y, cuando aplica, la relación; el llamador debe abortar
// el arranque ante ese error.
func cargarServicios(dir string, urls ServiceURLs) (map[string]servicioCargado, error) {
	patron := filepath.Join(dir, "relaciones_*.yaml")
	rutas, err := filepath.Glob(patron)
	if err != nil {
		return nil, fmt.Errorf("no se pudo listar archivos de relaciones en %q: %w", dir, err)
	}
	sort.Strings(rutas)

	configs := make(map[string]servicioCargado, len(rutas))
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
		configs[sc.Service] = servicioCargado{ruta: ruta, sc: sc}
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

	// El bloque write de la raíz, si se declara, no propaga llave de un padre.
	if sc.Write != nil {
		if err := validarWriteConfig(ruta, sc.Service, "(raíz)", *sc.Write, true); err != nil {
			return err
		}
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

	// Coherencia del bloque write de la relación, si se declara. Una relación
	// escribible no raíz debe propagar la llave del padre (fk, shared_pk o
	// fk_inbound) y, en consecuencia, debe identificar de forma determinista su
	// campo relacional derivado. Un modo cuyo campo derivado no puede resolverse
	// (por ejemplo fk sin related_field o fk_inbound sin local_field) es una
	// relación incoherente que se rechaza al cargar.
	if rel.Write != nil {
		if err := validarWriteConfig(ruta, servicio, id, *rel.Write, false); err != nil {
			return err
		}
		if _, ok := rel.CampoDerivado(); !ok {
			return fallo(fmt.Sprintf(
				"write key_propagation %q no identifica un campo derivado determinista "+
					"(fk requiere related_field, fk_inbound requiere local_field)",
				strings.TrimSpace(rel.Write.KeyPropagation),
			))
		}
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

// validarWriteConfig valida la coherencia de un bloque write. esRaiz indica si
// la configuración corresponde a la entidad raíz del servicio, que no propaga la
// llave de un padre (su key_propagation debe quedar vacío); una relación hija
// escribible debe declarar un modo de propagación válido (fk o shared_pk).
// Verifica además que los campos inmutables y de clave natural estén contenidos
// en los campos escribibles. Aborta con un mensaje que identifica el archivo, el
// servicio y la relación afectada.
func validarWriteConfig(ruta, servicio, relacion string, wc WriteConfig, esRaiz bool) error {
	fallo := func(detalle string) error {
		return fmt.Errorf(
			"configuración de relaciones inválida en %q (servicio %q, relación %q): write %s",
			ruta, servicio, relacion, detalle,
		)
	}

	prop := strings.TrimSpace(wc.KeyPropagation)
	if esRaiz {
		if prop != "" {
			return fallo(fmt.Sprintf(
				"la entidad raíz no propaga llave de un padre; key_propagation debe quedar vacío, no %q", prop,
			))
		}
	} else {
		if prop != KeyPropagationFK && prop != KeyPropagationSharedPK && prop != KeyPropagationFKInbound {
			return fallo(fmt.Sprintf(
				"key_propagation debe ser %q, %q o %q",
				KeyPropagationFK, KeyPropagationSharedPK, KeyPropagationFKInbound,
			))
		}
	}

	escribibles := make(map[string]struct{}, len(wc.WritableFields))
	for _, campo := range wc.WritableFields {
		if strings.TrimSpace(campo) == "" {
			return fallo("writable_fields contiene un campo vacío")
		}
		escribibles[campo] = struct{}{}
	}

	for _, campo := range wc.ImmutableFields {
		if _, ok := escribibles[campo]; !ok {
			return fallo(fmt.Sprintf("immutable_fields contiene %q que no está en writable_fields", campo))
		}
	}

	for _, campo := range wc.NaturalKey {
		if _, ok := escribibles[campo]; !ok {
			return fallo(fmt.Sprintf("natural_key contiene %q que no está en writable_fields", campo))
		}
	}

	return nil
}

// MustLoadRelaciones carga y valida los archivos de relaciones y aborta el
// arranque del proceso si la configuración está ausente, es inválida o es
// inconsistente, imprimiendo el mensaje que identifica el archivo y la relación.
func MustLoadRelaciones(dir string, urls ServiceURLs) map[string]ServiceConfig {
	comp, err := CargarComposicion(dir, urls)
	if err != nil {
		log.Fatalf("no se pudo cargar la configuración de relaciones: %v", err)
	}
	for _, nombre := range comp.Huerfanos {
		log.Printf("objeto huérfano: %q está definido pero no lo referencia ningún servicio y no entra a ningún árbol efectivo", nombre)
	}
	return comp.Servicios
}

// MaxProfundidadComposicion es el número máximo de niveles de referencias anidadas
// en el árbol efectivo de un servicio. El primer nivel son las referencias
// declaradas por el servicio. Una referencia que quede en un nivel mayor aborta el
// arranque.
const MaxProfundidadComposicion = 8

// servicioCargado guarda la configuración de un servicio junto con la ruta de su
// archivo, para identificar el origen de los errores de composición.
type servicioCargado struct {
	ruta string
	sc   ServiceConfig
}

// Composicion es el resultado de cargar la configuración de relaciones con su
// composición resuelta. Servicios contiene el árbol efectivo de cada servicio, en la
// forma que consumen los motores. Huerfanos lista, ordenados por nombre, los
// objetos definidos que no forman parte del árbol de ningún servicio.
type Composicion struct {
	Servicios map[string]ServiceConfig
	Huerfanos []string
}

// CargarComposicion carga los servicios de config/relaciones_*.yaml y el catálogo de
// objetos de config/objetos_*.yaml, resuelve las referencias y aplana el árbol
// efectivo de cada servicio. Un error de carga, de ubicación de objeto o de
// composición devuelve un error que nombra el fragmento responsable.
func CargarComposicion(dir string, urls ServiceURLs) (Composicion, error) {
	servicios, err := cargarServicios(dir, urls)
	if err != nil {
		return Composicion{}, err
	}

	catalogo, err := cargarCatalogoObjetos(dir, urls)
	if err != nil {
		return Composicion{}, err
	}
	if err := validarReferenciasColgantes(servicios, catalogo); err != nil {
		return Composicion{}, err
	}
	if err := validarCiclosObjetos(catalogo); err != nil {
		return Composicion{}, err
	}

	nombres := make([]string, 0, len(servicios))
	for nombre := range servicios {
		nombres = append(nombres, nombre)
	}
	sort.Strings(nombres)

	efectivos := make(map[string]ServiceConfig, len(servicios))
	for _, nombre := range nombres {
		cargado := servicios[nombre]
		sc, err := componerServicio(cargado, catalogo, urls)
		if err != nil {
			return Composicion{}, err
		}
		efectivos[nombre] = sc
	}

	return Composicion{Servicios: efectivos, Huerfanos: calcularHuerfanos(servicios, catalogo)}, nil
}

// cargarCatalogoObjetos lee y valida cada config/objetos_*.yaml y construye el
// catálogo. Cada objeto debe declarar la ubicación de su CRUD.
func cargarCatalogoObjetos(dir string, urls ServiceURLs) (CatalogoObjetos, error) {
	rutas, err := filepath.Glob(filepath.Join(dir, "objetos_*.yaml"))
	if err != nil {
		return CatalogoObjetos{}, fmt.Errorf("no se pudo listar archivos de objetos en %q: %w", dir, err)
	}
	sort.Strings(rutas)

	objetos := make([]ObjetoDireccionable, 0, len(rutas))
	for _, ruta := range rutas {
		contenido, err := os.ReadFile(ruta)
		if err != nil {
			return CatalogoObjetos{}, fmt.Errorf("no se pudo leer el archivo de objetos %q: %w", ruta, err)
		}
		obj, err := DecodificarObjetoDireccionable(contenido)
		if err != nil {
			return CatalogoObjetos{}, fmt.Errorf("archivo de objetos %q: %w", ruta, err)
		}
		if err := validarUbicacionObjeto(obj, urls); err != nil {
			return CatalogoObjetos{}, fmt.Errorf("archivo de objetos %q: %w", ruta, err)
		}
		objetos = append(objetos, obj)
	}

	return NuevoCatalogoObjetos(objetos)
}

// validarUbicacionObjeto comprueba que el objeto declare un CRUD existente en urls,
// una tabla y un endpoint, y que su natural_key sea un subconjunto de los campos
// writable.
func validarUbicacionObjeto(obj ObjetoDireccionable, urls ServiceURLs) error {
	if _, existe := urls[obj.RelatedAPI]; !existe {
		return fmt.Errorf("objeto %q: related_api %q no existe en la configuración de URLs", obj.Name, obj.RelatedAPI)
	}
	if strings.TrimSpace(obj.RelatedTable) == "" {
		return fmt.Errorf("objeto %q: related_table es obligatorio", obj.Name)
	}
	if strings.TrimSpace(obj.Endpoint) == "" {
		return fmt.Errorf("objeto %q: endpoint es obligatorio", obj.Name)
	}

	escribibles := make(map[string]struct{})
	for _, campo := range obj.Campos.CamposEscritura() {
		escribibles[campo] = struct{}{}
	}
	for _, campo := range obj.NaturalKey {
		if _, ok := escribibles[campo]; !ok {
			return fmt.Errorf("objeto %q: natural_key contiene %q que no está entre los campos writable", obj.Name, campo)
		}
	}
	return nil
}

// componerServicio resuelve las referencias del servicio, añade el árbol resultante
// a sus relaciones declaradas en línea y valida el árbol efectivo con las mismas
// reglas que una configuración en línea. Un servicio sin referencias se devuelve sin
// cambios.
func componerServicio(cargado servicioCargado, catalogo CatalogoObjetos, urls ServiceURLs) (ServiceConfig, error) {
	sc := cargado.sc
	if len(sc.Referencias) == 0 {
		return sc, nil
	}

	nodos, err := componerNodos(catalogo, sc.Referencias, 1, "servicio "+sc.Service)
	if err != nil {
		return ServiceConfig{}, fmt.Errorf("archivo de relaciones %q: %w", cargado.ruta, err)
	}

	efectivo := make([]RelationConfig, 0, len(sc.Relations)+len(nodos))
	efectivo = append(efectivo, sc.Relations...)
	sc.Relations = append(efectivo, nodos...)

	if err := validarServiceConfig(cargado.ruta, sc, urls); err != nil {
		return ServiceConfig{}, err
	}
	return sc, nil
}

// componerNodos convierte cada referencia en un nodo del árbol y compone de forma
// recursiva los hijos del objeto referenciado. nivel es la profundidad de las
// referencias dentro del árbol, empezando en 1. origen nombra al referenciador en
// los errores.
func componerNodos(catalogo CatalogoObjetos, refs []ReferenciaObjeto, nivel int, origen string) ([]RelationConfig, error) {
	nodos := make([]RelationConfig, 0, len(refs))
	for _, ref := range refs {
		if nivel > MaxProfundidadComposicion {
			return nil, fmt.Errorf(
				"profundidad de composición excedida en la referencia %q (%s): el máximo es %d niveles",
				ref.MappedName, origen, MaxProfundidadComposicion,
			)
		}
		if err := ref.Validar(); err != nil {
			return nil, fmt.Errorf("%s: %w", origen, err)
		}
		obj, ok := catalogo.Objeto(ref.Objeto)
		if !ok {
			return nil, fmt.Errorf("referencia colgante: el objeto %q no existe; lo referencia %s", ref.Objeto, origen)
		}
		hijos, err := componerNodos(catalogo, obj.Referencias, nivel+1, "objeto "+obj.Name)
		if err != nil {
			return nil, err
		}
		nodos = append(nodos, construirRelacion(obj, ref.Arista, hijos))
	}
	return nodos, nil
}

// servicioLogico devuelve la llave lógica de servicio del objeto para el nodo
// compuesto: el related_service declarado explícitamente, o la derivación del
// nombre cuando se omite. Es la llave con la que el registro related_service→
// resolver localiza el resolver del nodo, por lo que nunca es el nombre PascalCase.
func servicioLogico(obj ObjetoDireccionable) string {
	if explicito := strings.TrimSpace(obj.RelatedService); explicito != "" {
		return explicito
	}
	return aLlaveLogicaServicio(obj.Name)
}

// aLlaveLogicaServicio convierte un nombre PascalCase o camelCase en la llave
// lógica snake_case correspondiente (p. ej. TipoContribuyente ->
// tipo_contribuyente). Trata una secuencia de mayúsculas consecutivas como una
// sola sigla: inserta el separador en la frontera palabra→mayúscula (userID ->
// user_id) y en la frontera sigla→palabra (HTTPServer -> http_server), nunca entre
// las letras de una misma sigla. Un nombre ya en minúsculas se conserva.
func aLlaveLogicaServicio(nombre string) string {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return ""
	}
	runas := []rune(nombre)
	var b strings.Builder
	b.Grow(len(nombre) + 4)
	for i, r := range runas {
		if i > 0 && unicode.IsUpper(r) {
			anterior := runas[i-1]
			// Frontera palabra→mayúscula: la minúscula/dígito previo cierra una
			// palabra (userID -> user_id).
			limitePalabra := !unicode.IsUpper(anterior)
			// Frontera sigla→palabra: una mayúscula previa pertenece a una sigla y
			// la minúscula siguiente abre una palabra nueva (HTTPServer ->
			// http_server). Sin minúscula siguiente, la mayúscula continúa la sigla.
			limiteSigla := unicode.IsUpper(anterior) &&
				i+1 < len(runas) && unicode.IsLower(runas[i+1])
			if limitePalabra || limiteSigla {
				b.WriteByte('_')
			}
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// construirRelacion produce el RelationConfig que consumen los motores para un nodo:
// la arista de la referencia, la ubicación del CRUD y las vistas de lectura y
// escritura de la tabla de campos del objeto. El bloque write solo existe cuando la
// arista propaga llave o admite algún verbo de escritura. El QueryTemplate sigue la
// regla de las relaciones vigentes: consulta por related_field con la lista de llaves.
func construirRelacion(obj ObjetoDireccionable, arista Arista, hijos []RelationConfig) RelationConfig {
	rel := RelationConfig{
		Name:            obj.Name,
		RelatedService:  servicioLogico(obj),
		RelatedAPI:      obj.RelatedAPI,
		RelatedTable:    obj.RelatedTable,
		Direction:       arista.Direction,
		LocalField:      arista.LocalField,
		RelatedField:    arista.RelatedField,
		Cardinality:     arista.Cardinality,
		Depth:           arista.Depth,
		MappedName:      arista.MappedName,
		Endpoint:        obj.Endpoint,
		QueryTemplate:   arista.RelatedField + ":{ids}",
		AllowedFields:   obj.Campos.CamposLectura(),
		NestedRelations: hijos,
	}
	if arista.KeyPropagation != "" || arista.Policy.algunoAdmitido() {
		rel.Write = &WriteConfig{
			WritableFields:  obj.Campos.CamposEscritura(),
			ImmutableFields: obj.Campos.CamposInmutables(),
			NaturalKey:      obj.NaturalKey,
			KeyPropagation:  arista.KeyPropagation,
			Policy:          arista.Policy,
		}
	}
	return rel
}

// LoadRelaciones carga y valida la configuración de relaciones y devuelve el árbol
// efectivo de cada servicio. Los objetos huérfanos no se devuelven; use
// CargarComposicion para obtenerlos.
func LoadRelaciones(dir string, urls ServiceURLs) (map[string]ServiceConfig, error) {
	comp, err := CargarComposicion(dir, urls)
	if err != nil {
		return nil, err
	}
	return comp.Servicios, nil
}

// validarReferenciasColgantes comprueba que toda referencia de servicio u objeto
// apunte a un objeto definido. El error nombra el objeto ausente y el referenciador.
func validarReferenciasColgantes(servicios map[string]servicioCargado, catalogo CatalogoObjetos) error {
	nombresServicios := make([]string, 0, len(servicios))
	for nombre := range servicios {
		nombresServicios = append(nombresServicios, nombre)
	}
	sort.Strings(nombresServicios)

	for _, nombre := range nombresServicios {
		cargado := servicios[nombre]
		for _, ref := range cargado.sc.Referencias {
			if _, ok := catalogo.Objeto(ref.Objeto); !ok {
				return fmt.Errorf(
					"archivo de relaciones %q: referencia colgante: el objeto %q no existe; lo referencia el servicio %q (referencia %q)",
					cargado.ruta, ref.Objeto, nombre, ref.MappedName,
				)
			}
		}
	}

	for _, nombre := range catalogo.Nombres() {
		obj, _ := catalogo.Objeto(nombre)
		for _, ref := range obj.Referencias {
			if _, ok := catalogo.Objeto(ref.Objeto); !ok {
				return fmt.Errorf(
					"referencia colgante: el objeto %q no existe; lo referencia el objeto %q (referencia %q)",
					ref.Objeto, obj.Name, ref.MappedName,
				)
			}
		}
	}
	return nil
}

// validarCiclosObjetos recorre el grafo de referencias entre objetos y aborta ante
// el primer ciclo. El error nombra la referencia que cierra el ciclo y el camino
// completo.
func validarCiclosObjetos(catalogo CatalogoObjetos) error {
	const (
		sinVisitar = iota
		enCamino
		terminado
	)
	estado := make(map[string]int, len(catalogo.objetos))
	var camino []string

	var visitar func(nombre string) error
	visitar = func(nombre string) error {
		estado[nombre] = enCamino
		camino = append(camino, nombre)
		obj, _ := catalogo.Objeto(nombre)
		for _, ref := range obj.Referencias {
			switch estado[ref.Objeto] {
			case enCamino:
				return fmt.Errorf(
					"ciclo de referencias: la referencia %q del objeto %q apunta a %q, que ya está en el camino %s",
					ref.MappedName, nombre, ref.Objeto, formatearCamino(camino, ref.Objeto),
				)
			case sinVisitar:
				if _, ok := catalogo.Objeto(ref.Objeto); !ok {
					continue
				}
				if err := visitar(ref.Objeto); err != nil {
					return err
				}
			}
		}
		camino = camino[:len(camino)-1]
		estado[nombre] = terminado
		return nil
	}

	for _, nombre := range catalogo.Nombres() {
		if estado[nombre] != sinVisitar {
			continue
		}
		if err := visitar(nombre); err != nil {
			return err
		}
	}
	return nil
}

// formatearCamino devuelve el tramo del camino que va desde la primera aparición de
// cierre hasta el final, cerrado de nuevo en cierre. Ejemplo: "A -> B -> A".
func formatearCamino(camino []string, cierre string) string {
	inicio := 0
	for i, nombre := range camino {
		if nombre == cierre {
			inicio = i
			break
		}
	}
	partes := append(append([]string(nil), camino[inicio:]...), cierre)
	return strings.Join(partes, " -> ")
}

// calcularHuerfanos devuelve, ordenados por nombre, los objetos que no se alcanzan
// desde ninguna referencia de servicio. Un huérfano no entra al árbol efectivo de
// ningún servicio.
func calcularHuerfanos(servicios map[string]servicioCargado, catalogo CatalogoObjetos) []string {
	alcanzados := make(map[string]bool, len(catalogo.objetos))
	var pila []string
	for _, cargado := range servicios {
		for _, ref := range cargado.sc.Referencias {
			pila = append(pila, ref.Objeto)
		}
	}
	for len(pila) > 0 {
		nombre := pila[len(pila)-1]
		pila = pila[:len(pila)-1]
		if alcanzados[nombre] {
			continue
		}
		obj, ok := catalogo.Objeto(nombre)
		if !ok {
			continue
		}
		alcanzados[nombre] = true
		for _, ref := range obj.Referencias {
			pila = append(pila, ref.Objeto)
		}
	}

	huerfanos := []string{}
	for _, nombre := range catalogo.Nombres() {
		if !alcanzados[nombre] {
			huerfanos = append(huerfanos, nombre)
		}
	}
	return huerfanos
}
