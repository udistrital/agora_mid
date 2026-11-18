package services

import "github.com/udistrital/agora_mid/config"

// campoAutorCuerpo es el campo de auditoría de autor que el cliente envía en el
// cuerpo. Solo las entidades cuya configuración de escritura lo declara lo reciben.
const campoAutorCuerpo = "UsuarioModificacionId"

// propagarAutorDeRaiz devuelve una copia del cuerpo en la que el autor de la
// raíz se copia a cada objeto anidado que declara UsuarioModificacionId en su
// configuración de escritura y no trae un valor propio. El cuerpo original no se
// modifica. Si la raíz no trae autor, la copia es idéntica al original.
//
// Un valor nil cuenta como ausencia, igual que una FK nula en la escritura.
func propagarAutorDeRaiz(
	cuerpo map[string]any,
	relaciones []config.RelationConfig,
) map[string]any {
	autor, ok := cuerpo[campoAutorCuerpo]
	if !ok || autor == nil {
		return cuerpo
	}
	copia := copiaProfunda(cuerpo)
	propagarEnObjeto(copia, relaciones, autor)
	return copia
}

// propagarEnObjeto recorre las relaciones del objeto y asigna el autor a cada
// hijo que lo declara y carece de valor propio, en todos los niveles.
func propagarEnObjeto(objeto map[string]any, relaciones []config.RelationConfig, autor any) {
	for _, rel := range relaciones {
		valor, ok := objeto[rel.MappedName]
		if !ok {
			continue
		}
		for _, hijo := range objetosDe(valor) {
			if declaraAutor(rel) && hijo[campoAutorCuerpo] == nil {
				hijo[campoAutorCuerpo] = autor
			}
			propagarEnObjeto(hijo, rel.NestedRelations, autor)
		}
	}
}

// declaraAutor indica si la relación acepta UsuarioModificacionId en escritura.
func declaraAutor(rel config.RelationConfig) bool {
	if rel.Write == nil {
		return false
	}
	for _, campo := range rel.Write.WritableFields {
		if campo == campoAutorCuerpo {
			return true
		}
	}
	return false
}

// objetosDe devuelve los objetos de un valor que puede ser un objeto único o una
// lista de objetos. Los elementos que no son objetos se omiten.
func objetosDe(valor any) []map[string]any {
	switch v := valor.(type) {
	case map[string]any:
		return []map[string]any{v}
	case []any:
		var objetos []map[string]any
		for _, elem := range v {
			if m, ok := elem.(map[string]any); ok {
				objetos = append(objetos, m)
			}
		}
		return objetos
	}
	return nil
}

// copiaProfunda duplica los mapas y las listas del cuerpo para que la propagación
// no modifique el cuerpo recibido. Los escalares se copian por valor.
func copiaProfunda(v map[string]any) map[string]any {
	salida := make(map[string]any, len(v))
	for k, elem := range v {
		salida[k] = copiaValor(elem)
	}
	return salida
}

// copiaValor duplica recursivamente un valor del cuerpo JSON.
func copiaValor(valor any) any {
	switch v := valor.(type) {
	case map[string]any:
		return copiaProfunda(v)
	case []any:
		lista := make([]any, len(v))
		for i, elem := range v {
			lista[i] = copiaValor(elem)
		}
		return lista
	}
	return valor
}
