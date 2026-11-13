package generador

import (
	"fmt"
	"sort"
	"strings"
)

// crudPorEsquema indica el CRUD que almacena cada esquema del DBML. Las tablas de
// esquemas fuera de este mapa no generan objetos.
var crudPorEsquema = map[string]string{
	"proveedores": "gestion_personas_crud",
	"terceros":    "terceros",
	"parametros":  "parametros",
	"ubicaciones": "ubicaciones",
}

// rolPorOmision es el rol de un campo que ninguna entrada de field_roles cubre.
var rolPorOmision = RolCampo{Readable: true, Writable: true}

// resolverRol devuelve el rol de un campo según la precedencia de field_roles:
// esquema.tabla.columna, esquema.ANY.columna, ANY.tabla.columna y ANY.ANY.columna. La
// primera entrada encontrada prevalece; sin ninguna, el rol por omisión.
func resolverRol(roles map[string]RolCampo, esquema, tabla, columna string) RolCampo {
	candidatas := []string{
		esquema + "." + tabla + "." + columna,
		esquema + "." + comodinANY + "." + columna,
		comodinANY + "." + tabla + "." + columna,
		comodinANY + "." + comodinANY + "." + columna,
	}
	for _, c := range candidatas {
		if rol, ok := roles[c]; ok {
			return rol
		}
	}
	return rolPorOmision
}

// ConstruirObjetos genera un objeto por tabla de los CRUD mapeados. Las aristas de
// una raíz de servicio no se emiten en su objeto: salen en ConstruirServicios, y la
// raíz queda con referencias vacías. Cuando una FK aparece en expand y en fk_inbound,
// prevalece fk_inbound. Devuelve error ante entradas en conflicto o no válidas.
func ConstruirObjetos(m *Modelo, s *Sidecars) ([]ObjetoConfig, error) {
	objetos, _, err := construirTodo(m, s)
	return objetos, err
}

// construirTodo construye el mapa de objetos una sola vez y deriva de él los objetos
// y los archivos de servicio.
func construirTodo(m *Modelo, s *Sidecars) ([]ObjetoConfig, []ServicioConfig, error) {
	mapa, orden, err := construirMapa(m, s)
	if err != nil {
		return nil, nil, err
	}
	servicios, err := serviciosDesde(mapa, s)
	if err != nil {
		return nil, nil, err
	}
	objetos := objetosDesde(mapa, orden, s)
	if err := validarCiclos(objetos); err != nil {
		return nil, nil, err
	}
	return objetos, servicios, nil
}

// validarCiclos falla si los objetos emitidos forman un ciclo de referencias, con la
// misma regla que el cargador aplica al arranque: un objeto no puede alcanzarse a sí
// mismo siguiendo sus referencias. El error nombra el camino que cierra el ciclo.
func validarCiclos(objetos []ObjetoConfig) error {
	porNombre := make(map[string]ObjetoConfig, len(objetos))
	nombres := make([]string, 0, len(objetos))
	for _, o := range objetos {
		porNombre[o.Name] = o
		nombres = append(nombres, o.Name)
	}
	sort.Strings(nombres)

	const (
		sinVisitar = iota
		enCamino
		terminado
	)
	estado := map[string]int{}
	var camino []string
	var visitar func(nombre string) error
	visitar = func(nombre string) error {
		estado[nombre] = enCamino
		camino = append(camino, nombre)
		for _, a := range porNombre[nombre].Referencias {
			switch estado[a.Objeto] {
			case enCamino:
				return fmt.Errorf("ciclo de referencias: %s -> %s", strings.Join(camino, " -> "), a.Objeto)
			case sinVisitar:
				if err := visitar(a.Objeto); err != nil {
					return err
				}
			}
		}
		camino = camino[:len(camino)-1]
		estado[nombre] = terminado
		return nil
	}
	for _, n := range nombres {
		if estado[n] == sinVisitar {
			if err := visitar(n); err != nil {
				return err
			}
		}
	}
	return nil
}

// objetosDesde devuelve los objetos en orden de declaración, con las referencias de
// cada raíz de servicio vaciadas.
func objetosDesde(mapa map[string]*ObjetoConfig, orden []string, s *Sidecars) []ObjetoConfig {
	for raiz := range s.Service {
		if o, ok := mapa[raiz]; ok {
			o.Referencias = []AristaConfig{}
		}
	}
	resultado := make([]ObjetoConfig, 0, len(orden))
	for _, clave := range orden {
		resultado = append(resultado, *mapa[clave])
	}
	return resultado
}

// construirMapa construye todos los objetos con sus aristas, indexados por
// esquema.tabla, y devuelve además el orden de declaración de las tablas.
func construirMapa(m *Modelo, s *Sidecars) (map[string]*ObjetoConfig, []string, error) {
	expand := indexarEntradas(s.Expand)
	inbound := indexarEntradas(s.FKInbound)
	consulta := indexarEntradas(s.Lookup)
	blacklist := conjunto(s.Blacklist)
	for _, e := range append(append([]EntradaSidecar{}, s.Expand...), s.FKInbound...) {
		if blacklist[e.Clave] {
			return nil, nil, fmt.Errorf("sidecar %s: la entrada %q está en blacklist y también en expand o fk_inbound", archivoBlacklist, e.Clave)
		}
	}

	objetos := map[string]*ObjetoConfig{}
	orden := []string{}
	for _, t := range m.Tablas {
		api, mapeado := crudPorEsquema[t.Esquema]
		if !mapeado {
			continue
		}
		clave := t.Esquema + "." + t.Nombre
		o, err := construirCampos(t, api, blacklist, s.Write[clave], s.FieldRoles)
		if err != nil {
			return nil, nil, err
		}
		objetos[clave] = o
		orden = append(orden, clave)
	}

	for _, r := range m.Refs {
		hijoClave := r.Hijo.Clave()
		entrada, enExpand := expand[hijoClave]
		entradaInbound, enInbound := inbound[hijoClave]
		entradaConsulta, enLookup := consulta[hijoClave]
		var err error
		switch {
		case enLookup:
			err = agregarAristaConsulta(r, entradaConsulta, objetos)
		case enInbound:
			err = agregarArista(r, true, entradaInbound, objetos, s)
		case enExpand:
			err = agregarArista(r, false, entrada, objetos, s)
		}
		if err != nil {
			return nil, nil, err
		}
	}

	for _, clave := range orden {
		o := objetos[clave]
		sort.SliceStable(o.Referencias, func(i, j int) bool {
			return o.Referencias[i].MappedName < o.Referencias[j].MappedName
		})
		for i := 1; i < len(o.Referencias); i++ {
			if o.Referencias[i].MappedName == o.Referencias[i-1].MappedName {
				return nil, nil, fmt.Errorf("mapped_name %q duplicado en el objeto %s", o.Referencias[i].MappedName, o.Name)
			}
		}
	}
	return objetos, orden, nil
}

// construirCampos crea el objeto de una tabla con su tabla de campos y su clave
// natural, omitiendo las columnas en blacklist.
func construirCampos(t Tabla, api string, blacklist map[string]bool, pol PoliticaEscritura, roles map[string]RolCampo) (*ObjetoConfig, error) {
	o := &ObjetoConfig{
		Name:         pascal(t.Nombre),
		RelatedAPI:   api,
		RelatedTable: t.Nombre,
		Endpoint:     "/v1/" + t.Nombre,
		Referencias:  []AristaConfig{},
	}
	escribibles := map[string]bool{}
	for _, c := range t.Columnas {
		if blacklist[t.Esquema+"."+t.Nombre+"."+c.Nombre] {
			continue
		}
		rol := resolverRol(roles, t.Esquema, t.Nombre, c.Nombre)
		escribibles[pascal(c.Nombre)] = rol.Writable
		o.Campos = append(o.Campos, CampoConfig{
			Name:      pascal(c.Nombre),
			Type:      tiposAlmacenamiento[c.Tipo],
			Nullable:  !c.NoNula,
			Readable:  rol.Readable,
			Writable:  rol.Writable,
			Immutable: rol.Immutable,
		})
	}
	for _, col := range pol.NaturalKey {
		nombre := pascal(col)
		escribible, existe := escribibles[nombre]
		if !existe {
			return nil, fmt.Errorf("natural_key %q de %s no es un campo del objeto", col, o.Name)
		}
		if !escribible {
			return nil, fmt.Errorf("natural_key %q de %s debe ser escribible", col, o.Name)
		}
		o.NaturalKey = append(o.NaturalKey, nombre)
	}
	return o, nil
}

// agregarArista añade la arista que produce una FK en expand o fk_inbound. Una FK de
// fk_inbound va en el hijo y apunta al padre. Una FK de expand va en el padre y apunta
// al hijo, con cardinalidad many, salvo la PK compartida (`-`). La profundidad y el
// nombre explícito son los de la entrada del sidecar.
func agregarArista(r Referencia, esInbound bool, e EntradaSidecar, objetos map[string]*ObjetoConfig, s *Sidecars) error {
	padreClave := r.Padre.Esquema + "." + r.Padre.Tabla
	hijoClave := r.Hijo.Esquema + "." + r.Hijo.Tabla
	padre, okPadre := objetos[padreClave]
	hijo, okHijo := objetos[hijoClave]
	if !okPadre || !okHijo {
		return fmt.Errorf("la FK %s apunta a un objeto fuera de los CRUD mapeados", r.Hijo.Clave())
	}

	if esInbound {
		nombre, err := nombreArista(e, r, hijo, nombreSinSufijoID(r.Hijo.Columna))
		if err != nil {
			return err
		}
		hijo.Referencias = append(hijo.Referencias, AristaConfig{
			Objeto:         padre.Name,
			Direction:      "local_to_related",
			Cardinality:    "one",
			LocalField:     pascal(r.Hijo.Columna),
			RelatedField:   pascal(r.Padre.Columna),
			KeyPropagation: "fk_inbound",
			Policy:         politica(s.Write[padreClave]),
			Depth:          e.Depth,
			MappedName:     nombre,
		})
		return nil
	}

	cardinalidad, propagacion := "many", "fk"
	if r.Compartida {
		cardinalidad, propagacion = "one", "shared_pk"
	}
	nombre, err := nombreArista(e, r, padre, hijo.Name)
	if err != nil {
		return err
	}
	padre.Referencias = append(padre.Referencias, AristaConfig{
		Objeto:         hijo.Name,
		Direction:      "related_to_local",
		Cardinality:    cardinalidad,
		LocalField:     pascal(r.Padre.Columna),
		RelatedField:   pascal(r.Hijo.Columna),
		KeyPropagation: propagacion,
		Policy:         politica(s.Write[hijoClave]),
		Depth:          e.Depth,
		MappedName:     nombre,
		Flatten:        r.Compartida && contiene(s.Flatten, r.Hijo.Clave()),
	})
	return nil
}

// agregarAristaConsulta añade la arista que produce una FK en lookup: va en el objeto
// que porta la columna y apunta al referenciado, de solo lectura (sin propagación de
// llave ni verbos), sin importar la política de write del referenciado.
func agregarAristaConsulta(r Referencia, e EntradaSidecar, objetos map[string]*ObjetoConfig) error {
	padre, okPadre := objetos[r.Padre.Esquema+"."+r.Padre.Tabla]
	hijo, okHijo := objetos[r.Hijo.Esquema+"."+r.Hijo.Tabla]
	if !okPadre || !okHijo {
		return fmt.Errorf("la FK %s apunta a un objeto fuera de los CRUD mapeados", r.Hijo.Clave())
	}
	nombre, err := nombreArista(e, r, hijo, nombreSinSufijoID(r.Hijo.Columna))
	if err != nil {
		return err
	}
	hijo.Referencias = append(hijo.Referencias, AristaConfig{
		Objeto:       padre.Name,
		Direction:    "local_to_related",
		Cardinality:  "one",
		LocalField:   pascal(r.Hijo.Columna),
		RelatedField: pascal(r.Padre.Columna),
		Depth:        e.Depth,
		MappedName:   nombre,
	})
	return nil
}

// nombreArista devuelve el mapped_name de la arista: el explícito de la entrada o, si
// no lo hay, porOmision. Falla, nombrando la entrada, si el nombre coincide con un
// campo del objeto que lleva la arista, porque ocultaría el campo plano.
func nombreArista(e EntradaSidecar, r Referencia, portador *ObjetoConfig, porOmision string) (string, error) {
	nombre := porOmision
	if e.MappedName != "" {
		nombre = e.MappedName
	}
	for _, c := range portador.Campos {
		if c.Name == nombre {
			return "", fmt.Errorf("la entrada %q produce mapped_name %q, que coincide con un campo de %s; declare mapped_name", r.Hijo.Clave(), nombre, portador.Name)
		}
	}
	return nombre, nil
}

// nombreSinSufijoID convierte la columna portadora en PascalCase sin el sufijo _id.
func nombreSinSufijoID(columna string) string {
	return pascal(strings.TrimSuffix(columna, "_id"))
}

// politica convierte la entrada de write en los verbos de una arista.
func politica(p PoliticaEscritura) PoliticaVerbos {
	return PoliticaVerbos{Creatable: p.Creatable, Updatable: p.Updatable, Deletable: p.Deletable}
}

// indexarEntradas indexa las entradas de un sidecar por su clave de columna portadora.
func indexarEntradas(entradas []EntradaSidecar) map[string]EntradaSidecar {
	m := make(map[string]EntradaSidecar, len(entradas))
	for _, e := range entradas {
		m[e.Clave] = e
	}
	return m
}

// conjunto convierte una lista de entradas en un conjunto para búsquedas.
func conjunto(entradas []string) map[string]bool {
	c := make(map[string]bool, len(entradas))
	for _, e := range entradas {
		c[e] = true
	}
	return c
}

// contiene informa si la lista contiene el valor.
func contiene(valores []string, valor string) bool {
	for _, v := range valores {
		if v == valor {
			return true
		}
	}
	return false
}

// pascal convierte un identificador snake_case en PascalCase.
func pascal(snake string) string {
	partes := strings.Split(snake, "_")
	for i, p := range partes {
		if p != "" {
			partes[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(partes, "")
}
