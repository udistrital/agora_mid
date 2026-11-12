package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/udistrital/agora_mid/helper"
)

// campoEstadoProveedor es la relación de estados del proveedor en el cuerpo.
const campoEstadoProveedor = "EstadoProveedor"

// estadoRecord es el registro de estado_proveedor leído del CRUD dueño.
type estadoRecord struct {
	Id          int  `json:"Id"`
	ProveedorId int  `json:"ProveedorId"`
	EstadoId    int  `json:"EstadoId"`
	Activo      bool `json:"Activo"`
}

// componerTransicionEstado devuelve una copia del cuerpo con la transición de
// estado_proveedor compuesta. La escritura registra un solo estado: si su EstadoId
// difiere de los estados activos del proveedor, el nuevo se crea primero y cada
// activo previo se marca con _Eliminar después. Si coincide con el activo vigente,
// el intento se reconcilia con ese registro y no se crea ni da de baja nada.
//
// Un proveedor sin Id en el cuerpo es una creación: no tiene estados previos y el
// estado se crea tal cual. Sin el campo EstadoProveedor, el cuerpo no cambia.
func (s ProveedorWriteService) componerTransicionEstado(
	ctx context.Context,
	cuerpo map[string]any,
) (map[string]any, error) {
	valor, presente := cuerpo[campoEstadoProveedor]
	if !presente || valor == nil {
		return cuerpo, nil
	}
	lista, ok := valor.([]any)
	if !ok || len(lista) != 1 {
		return nil, &helper.ErrorEscritura{
			Path: campoEstadoProveedor, Motivo: "registre un solo estado por escritura",
		}
	}
	intento, ok := lista[0].(map[string]any)
	if !ok || intento["EstadoId"] == nil {
		return nil, &helper.ErrorEscritura{
			Path: campoEstadoProveedor + ".EstadoId", Motivo: "es obligatorio",
		}
	}

	copia := copiaProfunda(cuerpo)
	proveedorID := idEntero(copia["Id"])
	if proveedorID <= 0 {
		// Sin Id, el proveedor existente se busca por su TerceroId, igual que el motor.
		resuelto, err := s.proveedorPorTercero(ctx, idEntero(copia["TerceroId"]))
		if err != nil {
			return nil, err
		}
		proveedorID = resuelto
	}
	if proveedorID <= 0 {
		return copia, nil
	}

	activos, err := s.estadosActivos(ctx, proveedorID)
	if err != nil {
		return nil, err
	}
	if len(activos) > 1 {
		return nil, &helper.ErrorEscritura{
			Path: campoEstadoProveedor, Motivo: "el proveedor tiene más de un estado activo",
		}
	}

	intentoCopia := copia[campoEstadoProveedor].([]any)[0].(map[string]any)
	delete(intentoCopia, "Id")
	estadoNuevo := idEntero(intentoCopia["EstadoId"])

	// Un activo con el mismo EstadoId hace de no-op: el intento lo referencia.
	mismo := 0
	for _, a := range activos {
		if a.EstadoId == estadoNuevo {
			mismo = a.Id
			break
		}
	}
	salida := []any{intentoCopia}
	if mismo > 0 {
		intentoCopia["Id"] = float64(mismo)
	}
	for _, a := range activos {
		if a.Id == mismo {
			continue
		}
		salida = append(salida, map[string]any{"Id": float64(a.Id), "_Eliminar": true})
	}
	copia[campoEstadoProveedor] = salida
	return copia, nil
}

// estadosActivos lee del CRUD de gestión de personas los estados activos del
// proveedor dado. El filtro se aplica también en el cliente, por si el CRUD ignora
// la consulta.
func (s ProveedorWriteService) estadosActivos(ctx context.Context, proveedorID int) ([]estadoRecord, error) {
	base := strings.TrimRight(strings.TrimSpace(s.URLs[apiGestion]), "/")
	if base == "" {
		return nil, fmt.Errorf("falta la URL base de gestión de personas CRUD")
	}
	consulta := url.QueryEscape(fmt.Sprintf("ProveedorId:%d,Activo:true", proveedorID))
	endpoint := base + "/v1/estado_proveedor/?query=" + consulta

	var raw json.RawMessage
	if _, err := s.Client.Get(ctx, endpoint, &raw); err != nil {
		return nil, fmt.Errorf("consultando estados del proveedor: %w", err)
	}
	registros, err := decodeAPIList[estadoRecord](raw)
	if err != nil {
		return nil, fmt.Errorf("consultando estados del proveedor: %w", err)
	}

	var activos []estadoRecord
	for _, r := range registros {
		if r.Activo && r.ProveedorId == proveedorID && r.Id > 0 {
			activos = append(activos, r)
		}
	}
	return activos, nil
}

// proveedorRecord es el registro de proveedor leído del CRUD dueño para resolver su
// Id por TerceroId.
type proveedorRecord struct {
	Id        int  `json:"Id"`
	TerceroId int  `json:"TerceroId"`
	Activo    bool `json:"Activo"`
}

// proveedorPorTercero devuelve el Id del proveedor activo cuyo TerceroId es el dado,
// o 0 si no existe o el TerceroId no es válido. La consulta es la misma que usa el
// motor para la clave natural del proveedor.
func (s ProveedorWriteService) proveedorPorTercero(ctx context.Context, terceroID int) (int, error) {
	if terceroID <= 0 {
		return 0, nil
	}
	base := strings.TrimRight(strings.TrimSpace(s.URLs[apiGestion]), "/")
	if base == "" {
		return 0, fmt.Errorf("falta la URL base de gestión de personas CRUD")
	}
	consulta := url.QueryEscape(fmt.Sprintf("TerceroId:%d,Activo:true", terceroID))
	endpoint := base + "/v1/proveedor/?query=" + consulta

	var raw json.RawMessage
	if _, err := s.Client.Get(ctx, endpoint, &raw); err != nil {
		return 0, fmt.Errorf("consultando proveedor por tercero: %w", err)
	}
	registros, err := decodeAPIList[proveedorRecord](raw)
	if err != nil {
		return 0, fmt.Errorf("consultando proveedor por tercero: %w", err)
	}
	for _, r := range registros {
		if r.Activo && r.TerceroId == terceroID && r.Id > 0 {
			return r.Id, nil
		}
	}
	return 0, nil
}
