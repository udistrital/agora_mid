package services

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/udistrital/agora_mid/models"
)

type fakeCaracterizacion struct {
	mu          sync.Mutex
	terceros    map[int64]map[string]any
	info        map[int64]map[string]any
	proveedores map[int64]map[string]any
	naturales   map[int64]map[string]any
	perfiles    map[int64]map[string]any
	contactos   map[int64]map[string]any
	actividades map[int64]map[string]any
	siguiente   map[string]int64
}

func nuevoFakeCaracterizacion() *fakeCaracterizacion {
	return &fakeCaracterizacion{
		terceros:    map[int64]map[string]any{},
		info:        map[int64]map[string]any{},
		proveedores: map[int64]map[string]any{},
		naturales:   map[int64]map[string]any{},
		perfiles:    map[int64]map[string]any{},
		contactos:   map[int64]map[string]any{},
		actividades: map[int64]map[string]any{},
		siguiente: map[string]int64{
			"proveedor": 77,
			"info":      200,
			"perfil":    300,
			"contacto":  400,
			"actividad": 500,
		},
	}
}

func (f *fakeCaracterizacion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var cuerpo map[string]any
	if r.Body != nil {
		data := make([]byte, 0)
		buf := make([]byte, 1024)
		for {
			n, err := r.Body.Read(buf)
			if n > 0 {
				data = append(data, buf[:n]...)
			}
			if err != nil {
				break
			}
		}
		if len(data) > 0 {
			_ = json.Unmarshal(data, &cuerpo)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	path := strings.TrimSuffix(r.URL.Path, "/")
	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/v1/tercero/"):
		id := idDeRuta(path)
		registro, ok := f.terceros[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"Message":"no encontrado"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(registro)

	case r.Method == http.MethodGet && path == "/v1/info_complementaria_tercero":
		_ = json.NewEncoder(w).Encode(f.filtrar(f.info, r.URL.Query().Get("query")))

	case r.Method == http.MethodGet && strings.HasPrefix(path, "/v1/info_complementaria_tercero/"):
		id := idDeRuta(path)
		registro, ok := f.info[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"Message":"no encontrado"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(registro)

	case r.Method == http.MethodPost && path == "/v1/info_complementaria_tercero":
		id := f.siguiente["info"]
		f.siguiente["info"]++
		cuerpo["Id"] = id
		f.info[id] = cuerpo
		_ = json.NewEncoder(w).Encode(cuerpo)

	case r.Method == http.MethodPut && strings.HasPrefix(path, "/v1/info_complementaria_tercero/"):
		id := idDeRuta(path)
		cuerpo["Id"] = id
		f.info[id] = cuerpo
		_ = json.NewEncoder(w).Encode(cuerpo)

	case r.Method == http.MethodGet && path == "/v1/proveedor":
		_ = json.NewEncoder(w).Encode(f.filtrar(f.proveedores, r.URL.Query().Get("query")))

	case r.Method == http.MethodGet && strings.HasPrefix(path, "/v1/proveedor/"):
		id := idDeRuta(path)
		registro, ok := f.proveedores[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"Message":"no encontrado"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(registro)

	case r.Method == http.MethodPost && path == "/v1/proveedor":
		id := f.siguiente["proveedor"]
		f.siguiente["proveedor"]++
		cuerpo["Id"] = id
		f.proveedores[id] = cuerpo
		_ = json.NewEncoder(w).Encode(cuerpo)

	case r.Method == http.MethodPut && strings.HasPrefix(path, "/v1/proveedor/"):
		id := idDeRuta(path)
		cuerpo["Id"] = id
		f.proveedores[id] = cuerpo
		_ = json.NewEncoder(w).Encode(cuerpo)

	case r.Method == http.MethodGet && strings.HasPrefix(path, "/v1/proveedor_natural/"):
		id := idDeRuta(path)
		registro, ok := f.naturales[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"Message":"no encontrado"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(registro)

	case r.Method == http.MethodPost && path == "/v1/proveedor_natural":
		id := idDeMapa(cuerpo)
		cuerpo["Activo"] = true
		f.naturales[id] = cuerpo
		_ = json.NewEncoder(w).Encode(cuerpo)

	case r.Method == http.MethodPut && strings.HasPrefix(path, "/v1/proveedor_natural/"):
		id := idDeRuta(path)
		cuerpo["Id"] = id
		f.naturales[id] = cuerpo
		_ = json.NewEncoder(w).Encode(cuerpo)

	case r.Method == http.MethodGet && path == "/v1/perfil_proveedor":
		_ = json.NewEncoder(w).Encode(f.filtrar(f.perfiles, r.URL.Query().Get("query")))

	case r.Method == http.MethodPost && path == "/v1/perfil_proveedor":
		id := f.siguiente["perfil"]
		f.siguiente["perfil"]++
		cuerpo["Id"] = id
		cuerpo["Activo"] = true
		f.perfiles[id] = cuerpo
		_ = json.NewEncoder(w).Encode(cuerpo)

	case r.Method == http.MethodGet && path == "/v1/contacto":
		_ = json.NewEncoder(w).Encode(f.filtrar(f.contactos, r.URL.Query().Get("query")))

	case r.Method == http.MethodPost && path == "/v1/contacto":
		id := f.siguiente["contacto"]
		f.siguiente["contacto"]++
		cuerpo["Id"] = id
		cuerpo["Activo"] = true
		f.contactos[id] = cuerpo
		_ = json.NewEncoder(w).Encode(cuerpo)

	case r.Method == http.MethodGet && strings.HasPrefix(path, "/v1/contacto/"):
		id := idDeRuta(path)
		registro, ok := f.contactos[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"Message":"no encontrado"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(registro)

	case r.Method == http.MethodPut && strings.HasPrefix(path, "/v1/contacto/"):
		id := idDeRuta(path)
		cuerpo["Id"] = id
		f.contactos[id] = cuerpo
		_ = json.NewEncoder(w).Encode(cuerpo)

	case r.Method == http.MethodGet && path == "/v1/actividad_economica_proveedor":
		_ = json.NewEncoder(w).Encode(f.filtrar(f.actividades, r.URL.Query().Get("query")))

	case r.Method == http.MethodPost && path == "/v1/actividad_economica_proveedor":
		id := f.siguiente["actividad"]
		f.siguiente["actividad"]++
		cuerpo["Id"] = id
		cuerpo["Activo"] = true
		f.actividades[id] = cuerpo
		_ = json.NewEncoder(w).Encode(cuerpo)

	default:
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"Message":"ruta no simulada"}`))
	}
}

func (f *fakeCaracterizacion) filtrar(
	registros map[int64]map[string]any,
	query string,
) []map[string]any {
	filtros := map[string]string{}
	for _, parte := range strings.Split(query, ",") {
		clave, valor, ok := strings.Cut(parte, ":")
		if ok {
			filtros[strings.TrimSpace(clave)] = strings.TrimSpace(valor)
		}
	}
	ids := make([]int64, 0, len(registros))
	for id := range registros {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	salida := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		registro := registros[id]
		if coincideFiltros(registro, filtros) {
			salida = append(salida, registro)
		}
	}
	return salida
}

func coincideFiltros(registro map[string]any, filtros map[string]string) bool {
	for clave, esperado := range filtros {
		switch clave {
		case "TerceroId", "ProveedorId", "InfoComplementariaId", "PerfilId",
			"TipoCanalContactoId", "ActividadEconomicaId":
			if idDeRelacion(registro[clave]) != parseEntero(esperado) {
				return false
			}
		case "Activo":
			activo, _ := registro[clave].(bool)
			if strconv.FormatBool(activo) != esperado {
				return false
			}
		default:
			if cadenaCuenta(registro[clave]) != esperado {
				return false
			}
		}
	}
	return true
}

func parseEntero(valor string) int64 {
	id, _ := strconv.ParseInt(valor, 10, 64)
	return id
}

func idDeRuta(path string) int64 {
	partes := strings.Split(path, "/")
	id, _ := strconv.ParseInt(partes[len(partes)-1], 10, 64)
	return id
}

type catalogosCaracterizacionFake struct {
	infoInactiva map[int64]bool
	codigos      map[string]int64
	ambiguos     map[string]bool
	grupos       map[int64]string
	lugaresMalo  map[int64]bool
}

func (f *catalogosCaracterizacionFake) LugarExiste(_ context.Context, id int64) (bool, error) {
	return !f.lugaresMalo[id], nil
}

func (f *catalogosCaracterizacionFake) InfoComplementariaExiste(_ context.Context, id int64) (bool, error) {
	return !f.infoInactiva[id], nil
}

func (f *catalogosCaracterizacionFake) InfoComplementariaPorCodigo(
	_ context.Context,
	codigo string,
) (int64, bool, bool, error) {
	id, ok := f.codigos[codigo]
	return id, ok, f.ambiguos[codigo], nil
}

func (f *catalogosCaracterizacionFake) InfoComplementariaDeGrupo(
	_ context.Context,
	id int64,
	grupo string,
) (bool, error) {
	esperado, configurado := f.grupos[id]
	if !configurado {
		return true, nil
	}
	return esperado == grupo, nil
}

type parametrosCaracterizacionFake struct {
	noExiste map[int64]bool
	tipos    map[int64]string
}

func (f *parametrosCaracterizacionFake) ParametroExiste(_ context.Context, id int64) (bool, error) {
	return !f.noExiste[id], nil
}

func (f *parametrosCaracterizacionFake) ParametroDeTipo(
	_ context.Context,
	id int64,
	tipo string,
) (bool, error) {
	if f.noExiste[id] {
		return false, nil
	}
	if esperado, ok := f.tipos[id]; ok {
		return esperado == tipo, nil
	}
	return true, nil
}

func servicioCaracterizacionPrueba(
	t *testing.T,
	fake *fakeCaracterizacion,
	catalogos *catalogosCaracterizacionFake,
	parametros *parametrosCaracterizacionFake,
) CaracterizacionNaturalService {
	t.Helper()
	crudDoble(t, fake.ServeHTTP, fake.ServeHTTP)
	if catalogos == nil {
		catalogos = &catalogosCaracterizacionFake{}
	}
	if parametros == nil {
		parametros = &parametrosCaracterizacionFake{}
	}
	return NewCaracterizacionNaturalService(catalogos, parametros, 1, 1, "CIIU", "")
}

func entradaNaturalPrueba() models.RegistroCaracterizacionNatural {
	pais := int64(1)
	experiencia := int64(60)
	profesional := int64(36)
	descripcion := "Servicios de consultoría"
	canal := int64(1)
	nombre := "Ana Pérez"
	return models.RegistroCaracterizacionNatural{
		TerceroId: 101,
		InfoComplementariaTercero: []models.InfoComplementariaTerceroInput{
			{InfoComplementariaId: ptrInt64(12), Dato: json.RawMessage(`"Femenino"`)},
			{CodigoInfoComplementaria: ptrString("HIJOS"), Dato: json.RawMessage(`true`)},
			{CodigoInfoComplementaria: ptrString("MIGRANTE"), Dato: json.RawMessage(`false`)},
		},
		ProveedorNatural: &models.ProveedorNaturalInput{
			PaisNacimientoId:       &pais,
			ExperienciaLaboral:     &experiencia,
			ExperienciaProfesional: &profesional,
			ResponsableIva:         ptrBool(true),
		},
		PerfilProveedor: []models.PerfilProveedorInput{{PerfilId: 5}},
		Contacto: []models.ContactoProveedorInput{{
			TipoCanalContactoId: canal,
			Proposito:           "notificaciones",
			ValorEspecifico:     "ana@correo.co",
			NombreContacto:      &nombre,
		}},
		ActividadEconomicaProveedor: []models.ActividadEconomicaProveedorInput{{ActividadEconomicaId: 10}},
		DescripcionPortafolio:       &descripcion,
	}
}

func ptrInt64(v int64) *int64    { return &v }
func ptrBool(v bool) *bool       { return &v }
func ptrString(v string) *string { return &v }

func TestCaracterizacionNaturalCreaTodasLasSecciones(t *testing.T) {
	fake := nuevoFakeCaracterizacion()
	fake.terceros[101] = map[string]any{
		"Id": 101, "Activo": true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
	}
	catalogos := &catalogosCaracterizacionFake{
		codigos: map[string]int64{"DESCRIPCIÓN": 900, "HIJOS": 21},
	}
	servicio := servicioCaracterizacionPrueba(t, fake, catalogos, nil)

	resultado, err := servicio.Registrar(
		context.Background(), entradaNaturalPrueba(), 101, "create",
	)
	if err != nil {
		t.Fatalf("Registrar no debe fallar: %v", err)
	}
	if resultado.Parcial {
		t.Fatalf("no se esperaba parcial: %+v", resultado.Avisos)
	}
	if resultado.Data.ProveedorId != 77 {
		t.Fatalf("ProveedorId inesperado: %d", resultado.Data.ProveedorId)
	}
	if len(resultado.Data.InfoComplementariaTercero) != 3 {
		t.Fatalf("se esperaban 3 datos de info complementaria: %+v", resultado.Data.InfoComplementariaTercero)
	}
	if resultado.Data.ProveedorNatural == nil ||
		idDeRelacion(resultado.Data.ProveedorNatural["PaisNacimientoId"]) != 1 {
		t.Fatalf("ProveedorNatural inesperado: %+v", resultado.Data.ProveedorNatural)
	}
	if len(resultado.Data.PerfilProveedor) != 1 ||
		len(resultado.Data.Contacto) != 1 ||
		len(resultado.Data.ActividadEconomicaProveedor) != 1 {
		t.Fatalf("colecciones inesperadas: %+v", resultado.Data)
	}
	if fake.naturales[77] == nil || fake.contactos[400] == nil || fake.perfiles[300] == nil {
		t.Fatalf("no se materializaron las entidades: %+v", fake)
	}
}

func TestCaracterizacionNaturalActualizaParcialSinBorrar(t *testing.T) {
	fake := nuevoFakeCaracterizacion()
	fake.terceros[101] = map[string]any{
		"Id": 101, "Activo": true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
	}
	fake.proveedores[77] = map[string]any{
		"Id": 77, "TerceroId": 101, "TipoProveedor": 1, "Activo": true,
		"DescripcionPortafolio": "anterior",
	}
	fake.naturales[77] = map[string]any{
		"Id": 77, "Activo": true, "PaisNacimientoId": 1,
		"ExperienciaLaboral": 10, "ResponsableIva": false,
	}
	fake.contactos[400] = map[string]any{
		"Id": 400, "ProveedorId": map[string]any{"Id": 77},
		"TipoCanalContactoId": 1, "Proposito": "NOTIFICACIONES",
		"ValorEspecifico": "viejo@correo.co", "NombreContacto": "Nombre anterior",
		"Activo": true,
	}
	servicio := servicioCaracterizacionPrueba(t, fake,
		&catalogosCaracterizacionFake{codigos: map[string]int64{"DESCRIPCIÓN": 900}}, nil)

	responsable := true
	experiencia := int64(20)
	contacto := models.ContactoProveedorInput{
		TipoCanalContactoId: 1,
		Proposito:           "notificaciones",
		ValorEspecifico:     "nuevo@correo.co",
	}
	resultado, err := servicio.Registrar(
		context.Background(),
		models.RegistroCaracterizacionNatural{
			ProveedorNatural: &models.ProveedorNaturalInput{
				ResponsableIva:     &responsable,
				ExperienciaLaboral: &experiencia,
			},
			Contacto: []models.ContactoProveedorInput{contacto},
		},
		101, "update",
	)
	if err != nil {
		t.Fatalf("Registrar no debe fallar: %v", err)
	}
	if resultado.Parcial {
		t.Fatalf("no se esperaba parcial: %+v", resultado.Avisos)
	}
	natural := fake.naturales[77]
	if idDeRelacion(natural["PaisNacimientoId"]) != 1 {
		t.Errorf("PaisNacimientoId se perdió: %+v", natural)
	}
	if natural["ResponsableIva"] != true {
		t.Errorf("ResponsableIva no se actualizó: %+v", natural)
	}
	if natural["ExperienciaLaboral"] != float64(20) && natural["ExperienciaLaboral"] != int64(20) {
		t.Errorf("ExperienciaLaboral no se actualizó: %+v", natural)
	}
	contactoGuardado := fake.contactos[400]
	if contactoGuardado["ValorEspecifico"] != "nuevo@correo.co" {
		t.Errorf("ValorEspecifico no se actualizó: %+v", contactoGuardado)
	}
	if contactoGuardado["NombreContacto"] != "Nombre anterior" {
		t.Errorf("NombreContacto se sobrescribió: %+v", contactoGuardado)
	}
}

func TestCaracterizacionNaturalMezclaExtension(t *testing.T) {
	fake := nuevoFakeCaracterizacion()
	fake.terceros[101] = map[string]any{
		"Id": 101, "Activo": true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
	}
	fake.proveedores[77] = map[string]any{
		"Id": 77, "TerceroId": 101, "TipoProveedor": 1, "Activo": true,
	}
	fake.info[200] = map[string]any{
		"Id": 200, "TerceroId": map[string]any{"Id": 101},
		"InfoComplementariaId": map[string]any{"Id": 900},
		"Dato":                 `{"MIGRANTE":false,"PERSONA_PEP":true}`,
		"Activo":               true,
	}
	servicio := servicioCaracterizacionPrueba(t, fake,
		&catalogosCaracterizacionFake{codigos: map[string]int64{"DESCRIPCIÓN": 900}}, nil)

	resultado, err := servicio.Registrar(
		context.Background(),
		models.RegistroCaracterizacionNatural{
			InfoComplementariaTercero: []models.InfoComplementariaTerceroInput{
				{CodigoInfoComplementaria: ptrString("VICTIMA_CONFLICTO"), Dato: json.RawMessage(`true`)},
			},
		},
		101, "update",
	)
	if err != nil {
		t.Fatalf("Registrar no debe fallar: %v", err)
	}
	if resultado.Parcial {
		t.Fatalf("no se esperaba parcial: %+v", resultado.Avisos)
	}
	dato := fake.info[200]["Dato"].(string)
	var mapa map[string]any
	if err := json.Unmarshal([]byte(dato), &mapa); err != nil {
		t.Fatalf("dato no es JSON: %v", err)
	}
	if mapa["MIGRANTE"] != false || mapa["PERSONA_PEP"] != true || mapa["VICTIMA_CONFLICTO"] != true {
		t.Fatalf("mezcla de extensión inesperada: %+v", mapa)
	}
}

func TestCaracterizacionNaturalRechazaProveedorInactivo(t *testing.T) {
	fake := nuevoFakeCaracterizacion()
	fake.terceros[101] = map[string]any{
		"Id": 101, "Activo": true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
	}
	fake.proveedores[77] = map[string]any{
		"Id": 77, "TerceroId": 101, "TipoProveedor": 1, "Activo": false,
	}
	servicio := servicioCaracterizacionPrueba(t, fake, nil, nil)

	_, err := servicio.Registrar(
		context.Background(),
		models.RegistroCaracterizacionNatural{},
		101, "update",
	)
	var caracterizacionErr *CaracterizacionNaturalError
	if !asCaracterizacionError(err, &caracterizacionErr) || caracterizacionErr.StatusCode != 409 {
		t.Fatalf("se esperaba 409, obtuvo %v", err)
	}
}

func TestCaracterizacionNaturalRechazaPerfilFueraDeGrupo(t *testing.T) {
	fake := nuevoFakeCaracterizacion()
	fake.terceros[101] = map[string]any{
		"Id": 101, "Activo": true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
	}
	catalogos := &catalogosCaracterizacionFake{
		grupos: map[int64]string{5: "Grupo_6"},
	}
	servicio := servicioCaracterizacionPrueba(t, fake, catalogos, nil)

	_, err := servicio.Registrar(
		context.Background(),
		models.RegistroCaracterizacionNatural{
			PerfilProveedor: []models.PerfilProveedorInput{{PerfilId: 5}},
		},
		101, "create",
	)
	var caracterizacionErr *CaracterizacionNaturalError
	if !asCaracterizacionError(err, &caracterizacionErr) || caracterizacionErr.StatusCode != 400 {
		t.Fatalf("se esperaba 400, obtuvo %v", err)
	}
}

func TestCaracterizacionNaturalPutSinProveedorDevuelve404(t *testing.T) {
	fake := nuevoFakeCaracterizacion()
	fake.terceros[101] = map[string]any{
		"Id": 101, "Activo": true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
	}
	servicio := servicioCaracterizacionPrueba(t, fake, nil, nil)

	_, err := servicio.Registrar(
		context.Background(),
		models.RegistroCaracterizacionNatural{},
		101, "update",
	)
	var caracterizacionErr *CaracterizacionNaturalError
	if !asCaracterizacionError(err, &caracterizacionErr) || caracterizacionErr.StatusCode != 404 {
		t.Fatalf("se esperaba 404, obtuvo %v", err)
	}
}

func TestCaracterizacionNaturalParcialSinPaisNacimiento(t *testing.T) {
	fake := nuevoFakeCaracterizacion()
	fake.terceros[101] = map[string]any{
		"Id": 101, "Activo": true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
	}
	servicio := servicioCaracterizacionPrueba(t, fake, nil, nil)

	resultado, err := servicio.Registrar(
		context.Background(),
		models.RegistroCaracterizacionNatural{
			ProveedorNatural: &models.ProveedorNaturalInput{},
		},
		101, "create",
	)
	if err != nil {
		t.Fatalf("Registrar no debe fallar: %v", err)
	}
	if !resultado.Parcial || len(resultado.Avisos) == 0 {
		t.Fatalf("se esperaba parcial con aviso: %+v", resultado)
	}
	if resultado.Data.ProveedorId <= 0 {
		t.Fatalf("el proveedor debía quedar materializado: %+v", resultado.Data)
	}
}

func asCaracterizacionError(err error, destino **CaracterizacionNaturalError) bool {
	if err == nil {
		return false
	}
	caracterizacionErr, ok := err.(*CaracterizacionNaturalError)
	if !ok {
		return false
	}
	*destino = caracterizacionErr
	return true
}

func TestCaracterizacionNaturalMensajeSinHost(t *testing.T) {
	fake := nuevoFakeCaracterizacion()
	fake.terceros[101] = map[string]any{
		"Id": 101, "Activo": true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
	}
	servicio := servicioCaracterizacionPrueba(t, fake, nil, nil)
	_, err := servicio.Registrar(
		context.Background(),
		models.RegistroCaracterizacionNatural{},
		999, "update",
	)
	if err == nil {
		t.Fatal("se esperaba error")
	}
	if strings.Contains(err.Error(), "127.0.0.1") || strings.Contains(err.Error(), "http://") {
		t.Fatalf("el error no debe exponer el host: %v", err)
	}
}

func TestCaracterizacionNaturalActualizaInfoComplementariaPorID(t *testing.T) {
	fake := nuevoFakeCaracterizacion()
	fake.terceros[101] = map[string]any{
		"Id": 101, "Activo": true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
	}
	fake.proveedores[77] = map[string]any{
		"Id": 77, "TerceroId": 101, "TipoProveedor": 1, "Activo": true,
	}
	fake.info[200] = map[string]any{
		"Id": 200, "TerceroId": map[string]any{"Id": 101},
		"InfoComplementariaId": map[string]any{"Id": 12},
		"Dato":                 `"Viejo"`, "Activo": true,
	}
	servicio := servicioCaracterizacionPrueba(t, fake, nil, nil)

	_, err := servicio.Registrar(
		context.Background(),
		models.RegistroCaracterizacionNatural{
			InfoComplementariaTercero: []models.InfoComplementariaTerceroInput{{
				Id:                   ptrInt64(200),
				InfoComplementariaId: ptrInt64(12),
				Dato:                 json.RawMessage(`"Nuevo"`),
			}},
		},
		101, "update",
	)
	if err != nil {
		t.Fatalf("Registrar no debe fallar: %v", err)
	}
	if fake.info[200]["Dato"] != `"Nuevo"` {
		t.Fatalf("no se actualizó por Id: %+v", fake.info[200])
	}
}

func TestCaracterizacionNaturalRechazaInfoComplementariaDeOtroTercero(t *testing.T) {
	fake := nuevoFakeCaracterizacion()
	fake.terceros[101] = map[string]any{
		"Id": 101, "Activo": true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
	}
	fake.proveedores[77] = map[string]any{
		"Id": 77, "TerceroId": 101, "TipoProveedor": 1, "Activo": true,
	}
	fake.info[200] = map[string]any{
		"Id": 200, "TerceroId": map[string]any{"Id": 999},
		"InfoComplementariaId": map[string]any{"Id": 12},
		"Dato":                 `"Viejo"`, "Activo": true,
	}
	servicio := servicioCaracterizacionPrueba(t, fake, nil, nil)

	resultado, err := servicio.Registrar(
		context.Background(),
		models.RegistroCaracterizacionNatural{
			InfoComplementariaTercero: []models.InfoComplementariaTerceroInput{{
				Id:                   ptrInt64(200),
				InfoComplementariaId: ptrInt64(12),
				Dato:                 json.RawMessage(`"Nuevo"`),
			}},
		},
		101, "update",
	)
	if err != nil {
		t.Fatalf("Registrar no debe fallar: %v", err)
	}
	if !resultado.Parcial || len(resultado.Avisos) == 0 {
		t.Fatalf("se esperaba parcial por conflicto: %+v", resultado)
	}
	if fake.info[200]["Dato"] != `"Viejo"` {
		t.Fatalf("no debía modificar el registro ajeno: %+v", fake.info[200])
	}
}

func TestCaracterizacionNaturalMezclaDatoObjeto(t *testing.T) {
	fake := nuevoFakeCaracterizacion()
	fake.terceros[101] = map[string]any{
		"Id": 101, "Activo": true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
	}
	fake.proveedores[77] = map[string]any{
		"Id": 77, "TerceroId": 101, "TipoProveedor": 1, "Activo": true,
	}
	fake.info[200] = map[string]any{
		"Id": 200, "TerceroId": map[string]any{"Id": 101},
		"InfoComplementariaId": map[string]any{"Id": 14},
		"Dato":                 `{"Via":"Calle","DetalleVia":"10"}`, "Activo": true,
	}
	servicio := servicioCaracterizacionPrueba(t, fake, nil, nil)

	_, err := servicio.Registrar(
		context.Background(),
		models.RegistroCaracterizacionNatural{
			InfoComplementariaTercero: []models.InfoComplementariaTerceroInput{{
				InfoComplementariaId: ptrInt64(14),
				Dato:                 json.RawMessage(`{"NumeroCruce":"20"}`),
			}},
		},
		101, "update",
	)
	if err != nil {
		t.Fatalf("Registrar no debe fallar: %v", err)
	}
	var dato map[string]any
	if err := json.Unmarshal([]byte(fake.info[200]["Dato"].(string)), &dato); err != nil {
		t.Fatalf("dato inválido: %v", err)
	}
	if dato["Via"] != "Calle" || dato["DetalleVia"] != "10" || dato["NumeroCruce"] != "20" {
		t.Fatalf("no se mezcló el objeto: %+v", dato)
	}
}

func TestCaracterizacionNaturalMezclaObjetosAnidados(t *testing.T) {
	fake := nuevoFakeCaracterizacion()
	fake.terceros[101] = map[string]any{
		"Id": 101, "Activo": true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
	}
	fake.proveedores[77] = map[string]any{
		"Id": 77, "TerceroId": 101, "TipoProveedor": 1, "Activo": true,
	}
	fake.info[200] = map[string]any{
		"Id": 200, "TerceroId": map[string]any{"Id": 101},
		"InfoComplementariaId": map[string]any{"Id": 900},
		"Dato":                 `{"DIRECCION":{"Via":"Calle","DetalleVia":"10"},"GENERO":12}`, "Activo": true,
	}
	servicio := servicioCaracterizacionPrueba(t, fake,
		&catalogosCaracterizacionFake{codigos: map[string]int64{"DESCRIPCIÓN": 900}}, nil)

	_, err := servicio.Registrar(
		context.Background(),
		models.RegistroCaracterizacionNatural{
			InfoComplementariaTercero: []models.InfoComplementariaTerceroInput{
				{CodigoInfoComplementaria: ptrString("DIRECCION"), Dato: json.RawMessage(`{"NumeroCruce":"20"}`)},
			},
		},
		101, "update",
	)
	if err != nil {
		t.Fatalf("Registrar no debe fallar: %v", err)
	}
	var dato map[string]any
	if err := json.Unmarshal([]byte(fake.info[200]["Dato"].(string)), &dato); err != nil {
		t.Fatalf("dato inválido: %v", err)
	}
	direccion, _ := dato["DIRECCION"].(map[string]any)
	if direccion["Via"] != "Calle" || direccion["DetalleVia"] != "10" || direccion["NumeroCruce"] != "20" {
		t.Fatalf("no se mezcló el objeto anidado: %+v", direccion)
	}
	if dato["GENERO"] != float64(12) {
		t.Fatalf("se perdió GENERO: %+v", dato)
	}
}

func TestCaracterizacionNaturalValidaDatosSemanticos(t *testing.T) {
	fake := nuevoFakeCaracterizacion()
	fake.terceros[101] = map[string]any{
		"Id": 101, "Activo": true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
	}
	catalogos := &catalogosCaracterizacionFake{
		grupos:      map[int64]string{12: "Grupo_6"},
		lugaresMalo: map[int64]bool{999: true},
	}
	servicio := servicioCaracterizacionPrueba(t, fake, catalogos, nil)

	casos := []struct {
		nombre  string
		entrada models.InfoComplementariaTerceroInput
	}{
		{
			nombre: "género no numérico",
			entrada: models.InfoComplementariaTerceroInput{
				CodigoInfoComplementaria: ptrString("GENERO"),
				Dato:                     json.RawMessage(`"Femenino"`),
			},
		},
		{
			nombre: "hijos no booleano",
			entrada: models.InfoComplementariaTerceroInput{
				CodigoInfoComplementaria: ptrString("HIJOS"),
				Dato:                     json.RawMessage(`"si"`),
			},
		},
		{
			nombre: "lugar inexistente",
			entrada: models.InfoComplementariaTerceroInput{
				CodigoInfoComplementaria: ptrString("LUGAR_RESIDENCIA"),
				Dato:                     json.RawMessage(`999`),
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			_, err := servicio.Registrar(
				context.Background(),
				models.RegistroCaracterizacionNatural{
					InfoComplementariaTercero: []models.InfoComplementariaTerceroInput{caso.entrada},
				},
				101, "create",
			)
			var caracterizacionErr *CaracterizacionNaturalError
			if !asCaracterizacionError(err, &caracterizacionErr) || caracterizacionErr.StatusCode != 400 {
				t.Fatalf("se esperaba 400, obtuvo %v", err)
			}
		})
	}
}

func TestCaracterizacionNaturalAvisaCodigoAmbiguo(t *testing.T) {
	fake := nuevoFakeCaracterizacion()
	fake.terceros[101] = map[string]any{
		"Id": 101, "Activo": true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
	}
	catalogos := &catalogosCaracterizacionFake{
		codigos:  map[string]int64{"MIGRANTE": 444},
		ambiguos: map[string]bool{"MIGRANTE": true},
	}
	servicio := servicioCaracterizacionPrueba(t, fake, catalogos, nil)

	resultado, err := servicio.Registrar(
		context.Background(),
		models.RegistroCaracterizacionNatural{
			InfoComplementariaTercero: []models.InfoComplementariaTerceroInput{
				{CodigoInfoComplementaria: ptrString("MIGRANTE"), Dato: json.RawMessage(`true`)},
			},
		},
		101, "create",
	)
	if err != nil {
		t.Fatalf("Registrar no debe fallar: %v", err)
	}
	if resultado.Parcial {
		t.Fatalf("un aviso de ambigüedad no debe marcar parcial: %+v", resultado.Avisos)
	}
	if len(resultado.Avisos) != 1 || resultado.Avisos[0].Kind != "ambiguous_match" {
		t.Fatalf("se esperaba aviso ambiguous_match: %+v", resultado.Avisos)
	}
}

func TestCaracterizacionNaturalRegistrarPorProveedor(t *testing.T) {
	fake := nuevoFakeCaracterizacion()
	fake.terceros[101] = map[string]any{
		"Id": 101, "Activo": true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
	}
	fake.proveedores[77] = map[string]any{
		"Id": 77, "TerceroId": 101, "TipoProveedor": 1, "Activo": true,
		"DescripcionPortafolio": "anterior",
	}
	fake.naturales[77] = map[string]any{
		"Id": 77, "Activo": true, "PaisNacimientoId": 48,
		"ExperienciaLaboral": 10, "ResponsableIva": false,
	}
	servicio := servicioCaracterizacionPrueba(t, fake, nil, nil)

	experiencia := int64(20)
	responsable := true
	resultado, err := servicio.RegistrarPorProveedor(
		context.Background(),
		models.RegistroCaracterizacionNatural{
			ProveedorNatural: &models.ProveedorNaturalInput{
				ExperienciaLaboral: &experiencia,
				ResponsableIva:     &responsable,
			},
		},
		77, "update",
	)
	if err != nil {
		t.Fatalf("RegistrarPorProveedor no debe fallar: %v", err)
	}
	if resultado.Data.ProveedorId != 77 || resultado.Data.TerceroId != 101 {
		t.Fatalf("ids inesperados: %+v", resultado.Data)
	}
	natural := fake.naturales[77]
	if idDeRelacion(natural["PaisNacimientoId"]) != 48 {
		t.Errorf("PaisNacimientoId se perdió: %+v", natural)
	}
	if natural["ResponsableIva"] != true {
		t.Errorf("ResponsableIva no se actualizó: %+v", natural)
	}
}

func TestCaracterizacionNaturalRegistrarPorProveedorInexistente(t *testing.T) {
	fake := nuevoFakeCaracterizacion()
	fake.terceros[101] = map[string]any{
		"Id": 101, "Activo": true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
	}
	servicio := servicioCaracterizacionPrueba(t, fake, nil, nil)

	_, err := servicio.RegistrarPorProveedor(
		context.Background(), models.RegistroCaracterizacionNatural{}, 999, "update",
	)
	var caracterizacionErr *CaracterizacionNaturalError
	if !asCaracterizacionError(err, &caracterizacionErr) || caracterizacionErr.StatusCode != 404 {
		t.Fatalf("se esperaba 404, obtuvo %v", err)
	}
}

func TestCaracterizacionNaturalRegistrarPorProveedorInactivo(t *testing.T) {
	fake := nuevoFakeCaracterizacion()
	fake.terceros[101] = map[string]any{
		"Id": 101, "Activo": true,
		"TipoContribuyenteId": map[string]any{"Id": 1},
	}
	fake.proveedores[77] = map[string]any{
		"Id": 77, "TerceroId": 101, "TipoProveedor": 1, "Activo": false,
	}
	servicio := servicioCaracterizacionPrueba(t, fake, nil, nil)

	_, err := servicio.RegistrarPorProveedor(
		context.Background(), models.RegistroCaracterizacionNatural{}, 77, "update",
	)
	var caracterizacionErr *CaracterizacionNaturalError
	if !asCaracterizacionError(err, &caracterizacionErr) || caracterizacionErr.StatusCode != 409 {
		t.Fatalf("se esperaba 409, obtuvo %v", err)
	}
}
