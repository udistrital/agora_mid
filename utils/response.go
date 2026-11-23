// Package utils reúne componentes compartidos por la capa de servicios:
// el formato de respuesta estándar de la OAS y el cliente HTTP reutilizable
// para consumir los CRUD externos.
package utils

// Response es el formato de respuesta estándar de la OAS que devuelven los
// CRUD externos (el mismo de gestion_contractual_crud). La capa de servicios
// entrega este tipo a sus consumidores.
type Response struct {
	Success  bool        `json:"Success"`
	Status   int         `json:"Status"`
	Message  string      `json:"Message"`
	Data     interface{} `json:"Data"`
	Metadata *Metadata   `json:"Metadata,omitempty"`
}

// Metadata es la paginación que el CRUD devuelve en el listado (GetAll). En
// las respuestas de GetOne, Post y Put queda nil y se omite por omitempty.
type Metadata struct {
	Total           int64 `json:"total"`
	Limit           int64 `json:"limit"`
	Offset          int64 `json:"offset"`
	CurrentPage     int64 `json:"currentPage"`
	TotalPages      int64 `json:"totalPages"`
	HasNextPage     bool  `json:"hasNextPage"`
	HasPreviousPage bool  `json:"hasPreviousPage"`
}
