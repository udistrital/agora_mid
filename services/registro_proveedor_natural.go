package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/beego/beego/v2/core/logs"

	"github.com/udistrital/agora_mid/models"
)

// errRegistro es un error público de la coordinación del registro. Lleva la
// etapa fallida para el log y un mensaje neutro para el llamador; no incorpora
// host, puerto ni URL interna.
type errRegistro struct {
	Etapa   string
	Mensaje string
}

func (e *errRegistro) Error() string { return e.Mensaje }

// RegistrarProveedorNatural orquesta el registro de un proveedor persona
// natural sobre los CRUD de terceros y proveedores. Devuelve el contenido de
// `Data`, un indicador de si se creó al menos un registro (para el 201), y un
// error público con la etapa fallida cuando corresponde.
//
// Regla de resolución del tercero (decisión de negocio):
//   - La consulta de datos_identificacion activa es la fuente de verdad.
//   - [] o [{}] -> crear tercero nuevo (rama B).
//   - primer objeto no vacío -> leer el tercero por TerceroId.Id (rama A).
func RegistrarProveedorNatural(ctx context.Context, in models.RegistroProveedorNatural) (models.RegistroProveedorData, bool, error) {
	data := models.RegistroProveedorData{TipoProveedor: in.TipoProveedor}
	creado := false

	// 1. Consultar la identificación activa.
	filtros := url.Values{}
	filtros.Set("query", fmt.Sprintf("Numero:%s,TipoDocumentoId:%d,Activo:true", in.DatosIdentificacion.Numero, in.DatosIdentificacion.TipoDocumentoId))

	var lista []map[string]any
	ok, _, err := GetAll(ctx, "datos_identificacion", filtros, &lista)
	if !ok {
		return data, false, etapaError("consulta-identificacion", "No fue posible consultar la identificación del tercero", err)
	}

	ident := primeraCoincidencia(lista)
	if ident != nil {
		advertirVariasCoincidencias(lista, ident, in.DatosIdentificacion.Numero)
	}

	var terceroID int64

	if ident == nil {
		// Rama B: no hay identificación activa -> crear tercero e identificación.
		nuevoTercero, err := crearTercero(ctx, in)
		if err != nil {
			return data, false, err
		}
		creado = true
		data.DatosBasicos = nuevoTercero
		terceroID = idDe(nuevoTercero)
		if terceroID == 0 {
			return data, creado, etapaError("alta-tercero", "La respuesta de creación del tercero no trae un Id válido", nil)
		}

		nuevaIdent, err := crearIdentificacion(ctx, in, terceroID)
		if err != nil {
			return data, creado, err
		}
		data.DatosIdentificacion = nuevaIdent
	} else {
		// Rama A: existe identificación activa -> actualizar identificación y tercero.
		identID := idDe(ident)
		terceroID = terceroIDDe(ident)
		if identID == 0 || terceroID == 0 {
			return data, false, etapaError("resolucion-identificacion", "La identificación recuperada no trae Id o TerceroId.Id válidos", nil)
		}

		identActualizada, err := actualizarIdentificacion(ctx, in, ident, identID)
		if err != nil {
			return data, creado, err
		}
		data.DatosIdentificacion = identActualizada

		terceroActualizado, err := actualizarTercero(ctx, in, terceroID)
		if err != nil {
			return data, creado, err
		}
		data.DatosBasicos = terceroActualizado
	}

	// 4. Resolver el proveedor por tercero + TipoProveedor + Activo.
	provID, provCreado, err := resolverProveedor(ctx, terceroID, in.TipoProveedor)
	if err != nil {
		return data, creado, err
	}
	if provCreado {
		creado = true
	}
	data.ProveedorId = provID

	return data, creado, nil
}

// primeraCoincidencia aplica la regla: [] o [{}] -> nil; primer objeto no vacío
// -> ese objeto.
func primeraCoincidencia(lista []map[string]any) map[string]any {
	if len(lista) == 0 {
		return nil
	}
	primero := lista[0]
	if len(primero) == 0 {
		return nil
	}
	return primero
}

// advertirVariasCoincidencias emite un warning cuando hay más de una
// identificación activa, con los filtros, la cantidad y el Id seleccionado,
// enmascarando el número de documento.
func advertirVariasCoincidencias(lista []map[string]any, seleccionada map[string]any, numero string) {
	if len(lista) <= 1 {
		return
	}
	// Los filtros se reportan como nombres de campo, sin el valor crudo del
	// documento; el número va enmascarado.
	logs.Warning("servicio=registro_proveedor_natural etapa=identificacion-multiple filtros=Numero,TipoDocumentoId,Activo coincidencias=%d id_seleccionado=%d documento=%s",
		len(lista), idDe(seleccionada), enmascararDocumento(numero))
}

// enmascararDocumento oculta el número de documento dejando visibles los
// últimos cuatro caracteres.
func enmascararDocumento(numero string) string {
	n := len(numero)
	if n <= 4 {
		return "****"
	}
	return "****" + numero[n-4:]
}

// crearTercero da de alta el tercero con los DatosBasicos de la petición.
func crearTercero(ctx context.Context, in models.RegistroProveedorNatural) (map[string]any, error) {
	payload := map[string]any{
		"PrimerNombre":    in.DatosBasicos.PrimerNombre,
		"SegundoNombre":   in.DatosBasicos.SegundoNombre,
		"PrimerApellido":  in.DatosBasicos.PrimerApellido,
		"SegundoApellido": in.DatosBasicos.SegundoApellido,
		"FechaNacimiento": in.DatosBasicos.FechaNacimiento,
		"Activo":          true,
	}
	var resp map[string]any
	if ok, _, err := Post(ctx, "tercero", payload, &resp); !ok {
		return nil, etapaError("alta-tercero", "No fue posible crear el tercero", err)
	}
	return resp, nil
}

// actualizarTercero lee el tercero por Id y actualiza solo FechaNacimiento,
// conservando los nombres y apellidos recuperados.
func actualizarTercero(ctx context.Context, in models.RegistroProveedorNatural, terceroID int64) (map[string]any, error) {
	var actual map[string]any
	if ok, _, err := GetOne(ctx, fmt.Sprintf("tercero/%d", terceroID), &actual); !ok {
		return nil, etapaError("lectura-tercero", "No fue posible consultar el tercero", err)
	}
	actual["FechaNacimiento"] = in.DatosBasicos.FechaNacimiento
	var resp map[string]any
	if ok, _, err := Put(ctx, fmt.Sprintf("tercero/%d", terceroID), actual, &resp); !ok {
		return nil, etapaError("actualizacion-tercero", "No fue posible actualizar el tercero", err)
	}
	return resp, nil
}

// crearIdentificacion da de alta la identificación asociada al tercero.
func crearIdentificacion(ctx context.Context, in models.RegistroProveedorNatural, terceroID int64) (map[string]any, error) {
	payload := map[string]any{
		"Numero":           in.DatosIdentificacion.Numero,
		"TipoDocumentoId":  map[string]any{"Id": in.DatosIdentificacion.TipoDocumentoId},
		"CiudadExpedicion": in.DatosIdentificacion.CiudadExpedicion,
		"FechaExpedicion":  in.DatosIdentificacion.FechaExpedicion,
		"TerceroId":        map[string]any{"Id": terceroID},
		"Activo":           true,
	}
	var resp map[string]any
	if ok, _, err := Post(ctx, "datos_identificacion", payload, &resp); !ok {
		return nil, etapaError("alta-identificacion", "No fue posible crear la identificación", err)
	}
	return resp, nil
}

// actualizarIdentificacion parte del registro recuperado y reemplaza solo los
// campos enviados, conservando Id, la relación con el tercero y los campos
// requeridos. Ejecuta un único Put por el Id de la identificación.
func actualizarIdentificacion(ctx context.Context, in models.RegistroProveedorNatural, actual map[string]any, identID int64) (map[string]any, error) {
	payload := map[string]any{}
	for k, v := range actual {
		payload[k] = v
	}
	payload["Numero"] = in.DatosIdentificacion.Numero
	payload["TipoDocumentoId"] = map[string]any{"Id": in.DatosIdentificacion.TipoDocumentoId}
	payload["CiudadExpedicion"] = in.DatosIdentificacion.CiudadExpedicion
	payload["FechaExpedicion"] = in.DatosIdentificacion.FechaExpedicion

	var resp map[string]any
	if ok, _, err := Put(ctx, fmt.Sprintf("datos_identificacion/%d", identID), payload, &resp); !ok {
		return nil, etapaError("actualizacion-identificacion", "No fue posible actualizar la identificación", err)
	}
	return resp, nil
}

// resolverProveedor consulta el proveedor activo por tercero + TipoProveedor;
// reutiliza su Id si existe (sin escrituras) o lo crea si falta.
func resolverProveedor(ctx context.Context, terceroID, tipoProveedor int64) (int64, bool, error) {
	filtros := url.Values{}
	filtros.Set("query", fmt.Sprintf("TerceroId:%d,TipoProveedor:%d,Activo:true", terceroID, tipoProveedor))

	var lista []map[string]any
	if ok, _, err := GetAllProveedores(ctx, "proveedor", filtros, &lista); !ok {
		return 0, false, etapaError("consulta-proveedor", "No fue posible consultar el proveedor", err)
	}
	if prov := primeraCoincidencia(lista); prov != nil {
		if id := idDe(prov); id != 0 {
			return id, false, nil
		}
		return 0, false, etapaError("resolucion-proveedor", "El proveedor recuperado no trae un Id válido", nil)
	}

	payload := map[string]any{
		"TerceroId":             terceroID,
		"TipoProveedor":         tipoProveedor,
		"Activo":                true,
		"UsuarioModificacionId": terceroID,
	}
	var resp map[string]any
	if ok, _, err := PostProveedores(ctx, "proveedor", payload, &resp); !ok {
		return 0, false, etapaError("alta-proveedor", "No fue posible crear el proveedor", err)
	}
	id := idDe(resp)
	if id == 0 {
		return 0, true, etapaError("alta-proveedor", "La respuesta de creación del proveedor no trae un Id válido", nil)
	}
	return id, true, nil
}

// idDe lee el campo Id de un objeto JSON decodificado (number -> int64).
func idDe(obj map[string]any) int64 {
	return numDe(obj["Id"])
}

// terceroIDDe lee TerceroId.Id de un registro de identificación.
func terceroIDDe(ident map[string]any) int64 {
	rel, ok := ident["TerceroId"].(map[string]any)
	if !ok {
		return 0
	}
	return numDe(rel["Id"])
}

// numDe convierte un valor JSON numérico a int64 (JSON decodifica números como
// float64 por defecto).
func numDe(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	default:
		return 0
	}
}

// etapaError construye el error público de la coordinación registrando la
// etapa fallida y la causa técnica (sin exponerla en el mensaje público).
func etapaError(etapa, mensaje string, causa error) error {
	if causa != nil {
		logs.Error("servicio=registro_proveedor_natural etapa=%s causa=%v", etapa, causa)
	} else {
		logs.Error("servicio=registro_proveedor_natural etapa=%s", etapa)
	}
	return &errRegistro{Etapa: etapa, Mensaje: mensaje}
}

// EsErrorRegistro indica si err proviene de la coordinación del registro.
func EsErrorRegistro(err error) bool {
	var e *errRegistro
	return errors.As(err, &e)
}
