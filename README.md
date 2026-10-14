# agora_mid

Middleware de registro de personas naturales y jurídicas. El MID coordina los servicios propietarios de los datos; no se convierte en maestro de identidad ni replica campos entre CRUD.

## Especificaciones Técnicas

### Tecnologías y Versiones

- [Golang](https://github.com/udistrital/lineamientos_oas/blob/master/instalacion_de_herramientas/golang.md)
- [Beego](https://github.com/udistrital/lineamientos_oas/blob/master/instalacion_de_herramientas/beego.md)
- [UtilsOAS](https://github.com/udistrital/lineamientos_oas/blob/master/generacion_de_apis/control_error_json_crud.md)

### Variables de Entorno

Configurar antes de ejecutar (ver también `.env.example` y `conf/app.conf`).

```shell
AGORA_MID_API_NAME=[nombre de la API]
AGORA_MID_SERVICE_NAME=[nombre del servicio]
AGORA_MID_API_BASE_DIR=[directorio base del API con respecto a $GOPATH/src]
AGORA_MID_HTTP_PORT=[puerto HTTP del API]
AGORA_MID_RUN_MODE=[modo de ejecución: dev, prod, test]
AGORA_MID_TERCEROS_CRUD_URL=[URL base del CRUD de Terceros, sin el sufijo /v1]
AGORA_MID_GESTION_PERSONAS_CRUD_URL=[URL base del CRUD de Gestión de Personas, sin el sufijo /v1]
AGORA_MID_PARAMETROS_CRUD_URL=[URL base del CRUD de Parámetros, sin el sufijo /v1]
```

### Ejecución del Proyecto

#### Clonar el repositorio

```shell
mkdir -p $GOPATH/src/github.com/udistrital
cd $GOPATH/src/github.com/udistrital
git clone https://github.com/udistrital/agora_mid.git
cd agora_mid
```

#### Ejecución para Desarrollo

```shell
cp .env.example .env # Configurar las variables de entorno
set -a && source .env && set +a

go mod tidy
bee run
```

#### Ejecución con Docker

Compile el proyecto y la imagen:

```shell
go mod tidy
go build -o main
docker build -t agora_mid .
```

Inice un contenedor con la imagen y las variables de entorno:

```shell
cp .env.example .env # Configurar las variables de entorno
set -a && source .env && set +a
docker run --name agora_mid -p "$AGORA_MID_HTTP_PORT:$AGORA_MID_HTTP_PORT" --env-file .env agora_mid
```

### Ejecución Pruebas

Ejecutar en un entorno con la versión de Go declarada en `go.mod` y acceso a las dependencias:

```bash
go test ./...
go build ./...
```

## Estado CI

| Develop | Relese 0.0.1 | Master |
| -- | -- | -- |
| [![Build Status](https://hubci.portaloas.udistrital.edu.co/api/badges/udistrital/agora_mid/status.svg?ref=refs/heads/develop)](https://hubci.portaloas.udistrital.edu.co/udistrital/agora_mid/) | [![Build Status](https://hubci.portaloas.udistrital.edu.co/api/badges/udistrital/agora_mid/status.svg?ref=refs/heads/release/0.0.1)](https://hubci.portaloas.udistrital.edu.co/udistrital/agora_mid/) | [![Build Status](https://hubci.portaloas.udistrital.edu.co/api/badges/udistrital/agora_mid/status.svg)](https://hubci.portaloas.udistrital.edu.co/udistrital/agora_mid/) |

## Licencia

This file is part of agora_mid.

agora_mid is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

agora_mid is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with novedades_crud. If not, see [https://www.gnu.org/licenses/](https://www.gnu.org/licenses/).
