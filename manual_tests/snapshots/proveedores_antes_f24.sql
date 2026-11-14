--
-- PostgreSQL database dump
--

\restrict e3ODu8pVl9WlvyYLVV1RavpJhJvVp6AZJWuibrkdp6S5zHw8oZ3dgZ8P7U0ltUd

-- Dumped from database version 18.6 (Debian 18.6-1.pgdg13+2)
-- Dumped by pg_dump version 18.6 (Debian 18.6-1.pgdg13+2)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

ALTER TABLE IF EXISTS ONLY proveedores.titulo_proveedor_natural DROP CONSTRAINT IF EXISTS fk_titulo_proveedor_natural_proveedor;
ALTER TABLE IF EXISTS ONLY proveedores.representacion DROP CONSTRAINT IF EXISTS fk_representacion_proveedor;
ALTER TABLE IF EXISTS ONLY proveedores.proveedor_natural DROP CONSTRAINT IF EXISTS fk_proveedor_natural_proveedor;
ALTER TABLE IF EXISTS ONLY proveedores.proveedor_juridico DROP CONSTRAINT IF EXISTS fk_proveedor_juridico_proveedor;
ALTER TABLE IF EXISTS ONLY proveedores.persona_juridica_tipo_entidad DROP CONSTRAINT IF EXISTS fk_persona_juridica_tipo_entidad_proveedor_juridico;
ALTER TABLE IF EXISTS ONLY proveedores.perfil_proveedor DROP CONSTRAINT IF EXISTS fk_perfil_proveedor_proveedor;
ALTER TABLE IF EXISTS ONLY proveedores.estado_proveedor DROP CONSTRAINT IF EXISTS fk_estado_proveedor_proveedor;
ALTER TABLE IF EXISTS ONLY proveedores.documento DROP CONSTRAINT IF EXISTS fk_documento_proveedor;
ALTER TABLE IF EXISTS ONLY proveedores.cuenta_bancaria DROP CONSTRAINT IF EXISTS fk_cuenta_bancaria_proveedor;
ALTER TABLE IF EXISTS ONLY proveedores.cuenta_bancaria DROP CONSTRAINT IF EXISTS fk_cuenta_bancaria_contacto;
ALTER TABLE IF EXISTS ONLY proveedores.contacto DROP CONSTRAINT IF EXISTS fk_contacto_proveedor;
ALTER TABLE IF EXISTS ONLY proveedores.actividad_economica_proveedor DROP CONSTRAINT IF EXISTS fk_actividad_economica_proveedor_proveedor;
DROP INDEX IF EXISTS proveedores.pk_titulo_proveedor_natural;
DROP INDEX IF EXISTS proveedores.pk_representacion;
DROP INDEX IF EXISTS proveedores.pk_proveedor_natural;
DROP INDEX IF EXISTS proveedores.pk_proveedor_juridico;
DROP INDEX IF EXISTS proveedores.pk_proveedor;
DROP INDEX IF EXISTS proveedores.pk_persona_juridica_tipo_entidad;
DROP INDEX IF EXISTS proveedores.pk_perfil_proveedor;
DROP INDEX IF EXISTS proveedores.pk_estado_proveedor;
DROP INDEX IF EXISTS proveedores.pk_documento;
DROP INDEX IF EXISTS proveedores.pk_cuenta_bancaria;
DROP INDEX IF EXISTS proveedores.pk_contacto;
DROP INDEX IF EXISTS proveedores.pk_actividad_economica_proveedor;
DROP INDEX IF EXISTS proveedores.idx_titulo_proveedor_natural_usuario_modificacion_id;
DROP INDEX IF EXISTS proveedores.idx_titulo_proveedor_natural_proveedor_id;
DROP INDEX IF EXISTS proveedores.idx_titulo_proveedor_natural_perfil_id;
DROP INDEX IF EXISTS proveedores.idx_titulo_proveedor_natural_nucleo_basico_conocimiento_snies_i;
DROP INDEX IF EXISTS proveedores.idx_titulo_proveedor_natural_fecha_modificacion;
DROP INDEX IF EXISTS proveedores.idx_titulo_proveedor_natural_fecha_creacion;
DROP INDEX IF EXISTS proveedores.idx_titulo_proveedor_natural_activo;
DROP INDEX IF EXISTS proveedores.idx_representacion_usuario_modificacion_id;
DROP INDEX IF EXISTS proveedores.idx_representacion_representante_id;
DROP INDEX IF EXISTS proveedores.idx_representacion_proveedor_id;
DROP INDEX IF EXISTS proveedores.idx_representacion_fecha_modificacion;
DROP INDEX IF EXISTS proveedores.idx_representacion_fecha_creacion;
DROP INDEX IF EXISTS proveedores.idx_representacion_activo;
DROP INDEX IF EXISTS proveedores.idx_proveedor_usuario_modificacion_id;
DROP INDEX IF EXISTS proveedores.idx_proveedor_tipo_proveedor;
DROP INDEX IF EXISTS proveedores.idx_proveedor_tercero_id;
DROP INDEX IF EXISTS proveedores.idx_proveedor_natural_usuario_modificacion_id;
DROP INDEX IF EXISTS proveedores.idx_proveedor_natural_pais_nacimiento_id;
DROP INDEX IF EXISTS proveedores.idx_proveedor_natural_fecha_modificacion;
DROP INDEX IF EXISTS proveedores.idx_proveedor_natural_fecha_creacion;
DROP INDEX IF EXISTS proveedores.idx_proveedor_natural_activo;
DROP INDEX IF EXISTS proveedores.idx_proveedor_juridico_usuario_modificacion_id;
DROP INDEX IF EXISTS proveedores.idx_proveedor_juridico_tipo_conformacion_id;
DROP INDEX IF EXISTS proveedores.idx_proveedor_juridico_regimen_contributivo;
DROP INDEX IF EXISTS proveedores.idx_proveedor_juridico_procedencia_empresa_id;
DROP INDEX IF EXISTS proveedores.idx_proveedor_juridico_pais_origen_id;
DROP INDEX IF EXISTS proveedores.idx_proveedor_juridico_fecha_modificacion;
DROP INDEX IF EXISTS proveedores.idx_proveedor_juridico_fecha_creacion;
DROP INDEX IF EXISTS proveedores.idx_proveedor_juridico_ciudad_origen_id;
DROP INDEX IF EXISTS proveedores.idx_proveedor_juridico_activo;
DROP INDEX IF EXISTS proveedores.idx_proveedor_fecha_modificacion;
DROP INDEX IF EXISTS proveedores.idx_proveedor_fecha_creacion;
DROP INDEX IF EXISTS proveedores.idx_proveedor_activo;
DROP INDEX IF EXISTS proveedores.idx_persona_juridica_tipo_entidad_usuario_modificacion_id;
DROP INDEX IF EXISTS proveedores.idx_persona_juridica_tipo_entidad_tipo_entidad_id;
DROP INDEX IF EXISTS proveedores.idx_persona_juridica_tipo_entidad_persona_juridica_id;
DROP INDEX IF EXISTS proveedores.idx_persona_juridica_tipo_entidad_fecha_modificacion;
DROP INDEX IF EXISTS proveedores.idx_persona_juridica_tipo_entidad_fecha_creacion;
DROP INDEX IF EXISTS proveedores.idx_persona_juridica_tipo_entidad_activo;
DROP INDEX IF EXISTS proveedores.idx_perfil_proveedor_usuario_modificacion_id;
DROP INDEX IF EXISTS proveedores.idx_perfil_proveedor_proveedor_id;
DROP INDEX IF EXISTS proveedores.idx_perfil_proveedor_perfil_id;
DROP INDEX IF EXISTS proveedores.idx_perfil_proveedor_fecha_modificacion;
DROP INDEX IF EXISTS proveedores.idx_perfil_proveedor_fecha_creacion;
DROP INDEX IF EXISTS proveedores.idx_perfil_proveedor_activo;
DROP INDEX IF EXISTS proveedores.idx_estado_proveedor_usuario_modificacion_id;
DROP INDEX IF EXISTS proveedores.idx_estado_proveedor_proveedor_id;
DROP INDEX IF EXISTS proveedores.idx_estado_proveedor_fecha_modificacion;
DROP INDEX IF EXISTS proveedores.idx_estado_proveedor_fecha_creacion;
DROP INDEX IF EXISTS proveedores.idx_estado_proveedor_estado_id;
DROP INDEX IF EXISTS proveedores.idx_estado_proveedor_activo;
DROP INDEX IF EXISTS proveedores.idx_documento_usuario_modificacion_id;
DROP INDEX IF EXISTS proveedores.idx_documento_tipo_documento_id;
DROP INDEX IF EXISTS proveedores.idx_documento_proveedor_id;
DROP INDEX IF EXISTS proveedores.idx_documento_fecha_modificacion;
DROP INDEX IF EXISTS proveedores.idx_documento_fecha_creacion;
DROP INDEX IF EXISTS proveedores.idx_documento_activo;
DROP INDEX IF EXISTS proveedores.idx_cuenta_bancaria_usuario_modificacion_id;
DROP INDEX IF EXISTS proveedores.idx_cuenta_bancaria_tipo_cuenta_id;
DROP INDEX IF EXISTS proveedores.idx_cuenta_bancaria_proveedor_id;
DROP INDEX IF EXISTS proveedores.idx_cuenta_bancaria_fecha_modificacion;
DROP INDEX IF EXISTS proveedores.idx_cuenta_bancaria_fecha_creacion;
DROP INDEX IF EXISTS proveedores.idx_cuenta_bancaria_entidad_bancaria_id;
DROP INDEX IF EXISTS proveedores.idx_cuenta_bancaria_contacto_id;
DROP INDEX IF EXISTS proveedores.idx_cuenta_bancaria_activo;
DROP INDEX IF EXISTS proveedores.idx_contacto_usuario_modificacion_id;
DROP INDEX IF EXISTS proveedores.idx_contacto_tipo_canal_contacto_id;
DROP INDEX IF EXISTS proveedores.idx_contacto_proveedor_id;
DROP INDEX IF EXISTS proveedores.idx_contacto_fecha_modificacion;
DROP INDEX IF EXISTS proveedores.idx_contacto_fecha_creacion;
DROP INDEX IF EXISTS proveedores.idx_contacto_activo;
DROP INDEX IF EXISTS proveedores.idx_actividad_economica_proveedor_usuario_modificacion_id;
DROP INDEX IF EXISTS proveedores.idx_actividad_economica_proveedor_proveedor_id;
DROP INDEX IF EXISTS proveedores.idx_actividad_economica_proveedor_fecha_modificacion;
DROP INDEX IF EXISTS proveedores.idx_actividad_economica_proveedor_fecha_creacion;
DROP INDEX IF EXISTS proveedores.idx_actividad_economica_proveedor_activo;
DROP INDEX IF EXISTS proveedores.idx_actividad_economica_proveedor_actividad_economica_id;
ALTER TABLE IF EXISTS ONLY proveedores.titulo_proveedor_natural DROP CONSTRAINT IF EXISTS titulo_proveedor_natural_pkey;
ALTER TABLE IF EXISTS ONLY proveedores.representacion DROP CONSTRAINT IF EXISTS representacion_pkey;
ALTER TABLE IF EXISTS ONLY proveedores.proveedor DROP CONSTRAINT IF EXISTS proveedor_pkey;
ALTER TABLE IF EXISTS ONLY proveedores.proveedor_natural DROP CONSTRAINT IF EXISTS proveedor_natural_pkey;
ALTER TABLE IF EXISTS ONLY proveedores.proveedor_juridico DROP CONSTRAINT IF EXISTS proveedor_juridico_pkey;
ALTER TABLE IF EXISTS ONLY proveedores.persona_juridica_tipo_entidad DROP CONSTRAINT IF EXISTS persona_juridica_tipo_entidad_pkey;
ALTER TABLE IF EXISTS ONLY proveedores.perfil_proveedor DROP CONSTRAINT IF EXISTS perfil_proveedor_pkey;
ALTER TABLE IF EXISTS ONLY proveedores.estado_proveedor DROP CONSTRAINT IF EXISTS estado_proveedor_pkey;
ALTER TABLE IF EXISTS ONLY proveedores.documento DROP CONSTRAINT IF EXISTS documento_pkey;
ALTER TABLE IF EXISTS ONLY proveedores.cuenta_bancaria DROP CONSTRAINT IF EXISTS cuenta_bancaria_pkey;
ALTER TABLE IF EXISTS ONLY proveedores.contacto DROP CONSTRAINT IF EXISTS contacto_pkey;
ALTER TABLE IF EXISTS ONLY proveedores.actividad_economica_proveedor DROP CONSTRAINT IF EXISTS actividad_economica_proveedor_pkey;
ALTER TABLE IF EXISTS proveedores.titulo_proveedor_natural ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS proveedores.representacion ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS proveedores.proveedor_natural ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS proveedores.proveedor_juridico ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS proveedores.proveedor ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS proveedores.persona_juridica_tipo_entidad ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS proveedores.perfil_proveedor ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS proveedores.estado_proveedor ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS proveedores.documento ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS proveedores.cuenta_bancaria ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS proveedores.contacto ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS proveedores.actividad_economica_proveedor ALTER COLUMN id DROP DEFAULT;
DROP SEQUENCE IF EXISTS proveedores.titulo_proveedor_natural_id_seq;
DROP TABLE IF EXISTS proveedores.titulo_proveedor_natural;
DROP SEQUENCE IF EXISTS proveedores.representacion_id_seq;
DROP TABLE IF EXISTS proveedores.representacion;
DROP SEQUENCE IF EXISTS proveedores.proveedor_natural_id_seq;
DROP TABLE IF EXISTS proveedores.proveedor_natural;
DROP SEQUENCE IF EXISTS proveedores.proveedor_juridico_id_seq;
DROP TABLE IF EXISTS proveedores.proveedor_juridico;
DROP SEQUENCE IF EXISTS proveedores.proveedor_id_seq;
DROP TABLE IF EXISTS proveedores.proveedor;
DROP SEQUENCE IF EXISTS proveedores.persona_juridica_tipo_entidad_id_seq;
DROP TABLE IF EXISTS proveedores.persona_juridica_tipo_entidad;
DROP SEQUENCE IF EXISTS proveedores.perfil_proveedor_id_seq;
DROP TABLE IF EXISTS proveedores.perfil_proveedor;
DROP SEQUENCE IF EXISTS proveedores.estado_proveedor_id_seq;
DROP TABLE IF EXISTS proveedores.estado_proveedor;
DROP SEQUENCE IF EXISTS proveedores.documento_id_seq;
DROP TABLE IF EXISTS proveedores.documento;
DROP SEQUENCE IF EXISTS proveedores.cuenta_bancaria_id_seq;
DROP TABLE IF EXISTS proveedores.cuenta_bancaria;
DROP SEQUENCE IF EXISTS proveedores.contacto_id_seq;
DROP TABLE IF EXISTS proveedores.contacto;
DROP SEQUENCE IF EXISTS proveedores.actividad_economica_proveedor_id_seq;
DROP TABLE IF EXISTS proveedores.actividad_economica_proveedor;
DROP SCHEMA IF EXISTS proveedores;
--
-- Name: proveedores; Type: SCHEMA; Schema: -; Owner: postgres
--

CREATE SCHEMA proveedores;


ALTER SCHEMA proveedores OWNER TO postgres;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: actividad_economica_proveedor; Type: TABLE; Schema: proveedores; Owner: postgres
--

CREATE TABLE proveedores.actividad_economica_proveedor (
    id integer NOT NULL,
    proveedor_id integer NOT NULL,
    actividad_economica_id integer NOT NULL,
    activo boolean DEFAULT true NOT NULL,
    fecha_creacion timestamp without time zone DEFAULT now() NOT NULL,
    fecha_modificacion timestamp without time zone,
    usuario_modificacion_id integer
);


ALTER TABLE proveedores.actividad_economica_proveedor OWNER TO postgres;

--
-- Name: COLUMN actividad_economica_proveedor.usuario_modificacion_id; Type: COMMENT; Schema: proveedores; Owner: postgres
--

COMMENT ON COLUMN proveedores.actividad_economica_proveedor.usuario_modificacion_id IS 'Posible referencia externa a Terceros CRUD';


--
-- Name: actividad_economica_proveedor_id_seq; Type: SEQUENCE; Schema: proveedores; Owner: postgres
--

CREATE SEQUENCE proveedores.actividad_economica_proveedor_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE proveedores.actividad_economica_proveedor_id_seq OWNER TO postgres;

--
-- Name: actividad_economica_proveedor_id_seq; Type: SEQUENCE OWNED BY; Schema: proveedores; Owner: postgres
--

ALTER SEQUENCE proveedores.actividad_economica_proveedor_id_seq OWNED BY proveedores.actividad_economica_proveedor.id;


--
-- Name: contacto; Type: TABLE; Schema: proveedores; Owner: postgres
--

CREATE TABLE proveedores.contacto (
    id integer NOT NULL,
    nombre_contacto character varying(100),
    valor_especifico character varying(100) NOT NULL,
    proveedor_id integer NOT NULL,
    tipo_canal_contacto_id integer NOT NULL,
    proposito character varying(50) NOT NULL,
    extension character varying(10),
    activo boolean DEFAULT true NOT NULL,
    fecha_creacion timestamp without time zone DEFAULT now() NOT NULL,
    fecha_modificacion timestamp without time zone,
    usuario_modificacion_id integer
);


ALTER TABLE proveedores.contacto OWNER TO postgres;

--
-- Name: COLUMN contacto.usuario_modificacion_id; Type: COMMENT; Schema: proveedores; Owner: postgres
--

COMMENT ON COLUMN proveedores.contacto.usuario_modificacion_id IS 'Posible referencia externa a Terceros CRUD';


--
-- Name: contacto_id_seq; Type: SEQUENCE; Schema: proveedores; Owner: postgres
--

CREATE SEQUENCE proveedores.contacto_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE proveedores.contacto_id_seq OWNER TO postgres;

--
-- Name: contacto_id_seq; Type: SEQUENCE OWNED BY; Schema: proveedores; Owner: postgres
--

ALTER SEQUENCE proveedores.contacto_id_seq OWNED BY proveedores.contacto.id;


--
-- Name: cuenta_bancaria; Type: TABLE; Schema: proveedores; Owner: postgres
--

CREATE TABLE proveedores.cuenta_bancaria (
    id integer NOT NULL,
    tipo_cuenta_id integer NOT NULL,
    numero_cuenta character varying(50) NOT NULL,
    entidad_bancaria_id integer NOT NULL,
    contacto_id integer,
    monto_capital_autorizado numeric(20,7),
    proveedor_id integer NOT NULL,
    activo boolean DEFAULT true NOT NULL,
    fecha_creacion timestamp without time zone DEFAULT now() NOT NULL,
    fecha_modificacion timestamp without time zone,
    usuario_modificacion_id integer
);


ALTER TABLE proveedores.cuenta_bancaria OWNER TO postgres;

--
-- Name: COLUMN cuenta_bancaria.usuario_modificacion_id; Type: COMMENT; Schema: proveedores; Owner: postgres
--

COMMENT ON COLUMN proveedores.cuenta_bancaria.usuario_modificacion_id IS 'Posible referencia externa a Terceros CRUD';


--
-- Name: cuenta_bancaria_id_seq; Type: SEQUENCE; Schema: proveedores; Owner: postgres
--

CREATE SEQUENCE proveedores.cuenta_bancaria_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE proveedores.cuenta_bancaria_id_seq OWNER TO postgres;

--
-- Name: cuenta_bancaria_id_seq; Type: SEQUENCE OWNED BY; Schema: proveedores; Owner: postgres
--

ALTER SEQUENCE proveedores.cuenta_bancaria_id_seq OWNED BY proveedores.cuenta_bancaria.id;


--
-- Name: documento; Type: TABLE; Schema: proveedores; Owner: postgres
--

CREATE TABLE proveedores.documento (
    id integer NOT NULL,
    tipo_documento_id integer NOT NULL,
    enlace uuid NOT NULL,
    proveedor_id integer NOT NULL,
    activo boolean DEFAULT true NOT NULL,
    fecha_creacion timestamp without time zone DEFAULT now() NOT NULL,
    fecha_modificacion timestamp without time zone,
    usuario_modificacion_id integer
);


ALTER TABLE proveedores.documento OWNER TO postgres;

--
-- Name: COLUMN documento.usuario_modificacion_id; Type: COMMENT; Schema: proveedores; Owner: postgres
--

COMMENT ON COLUMN proveedores.documento.usuario_modificacion_id IS 'Posible referencia externa a Terceros CRUD';


--
-- Name: documento_id_seq; Type: SEQUENCE; Schema: proveedores; Owner: postgres
--

CREATE SEQUENCE proveedores.documento_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE proveedores.documento_id_seq OWNER TO postgres;

--
-- Name: documento_id_seq; Type: SEQUENCE OWNED BY; Schema: proveedores; Owner: postgres
--

ALTER SEQUENCE proveedores.documento_id_seq OWNED BY proveedores.documento.id;


--
-- Name: estado_proveedor; Type: TABLE; Schema: proveedores; Owner: postgres
--

CREATE TABLE proveedores.estado_proveedor (
    id integer NOT NULL,
    estado_id integer,
    proveedor_id integer NOT NULL,
    observacion character varying(100),
    activo boolean DEFAULT true NOT NULL,
    fecha_creacion timestamp without time zone DEFAULT now() NOT NULL,
    fecha_modificacion timestamp without time zone,
    usuario_modificacion_id integer
);


ALTER TABLE proveedores.estado_proveedor OWNER TO postgres;

--
-- Name: COLUMN estado_proveedor.usuario_modificacion_id; Type: COMMENT; Schema: proveedores; Owner: postgres
--

COMMENT ON COLUMN proveedores.estado_proveedor.usuario_modificacion_id IS 'Posible referencia externa a Terceros CRUD';


--
-- Name: estado_proveedor_id_seq; Type: SEQUENCE; Schema: proveedores; Owner: postgres
--

CREATE SEQUENCE proveedores.estado_proveedor_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE proveedores.estado_proveedor_id_seq OWNER TO postgres;

--
-- Name: estado_proveedor_id_seq; Type: SEQUENCE OWNED BY; Schema: proveedores; Owner: postgres
--

ALTER SEQUENCE proveedores.estado_proveedor_id_seq OWNED BY proveedores.estado_proveedor.id;


--
-- Name: perfil_proveedor; Type: TABLE; Schema: proveedores; Owner: postgres
--

CREATE TABLE proveedores.perfil_proveedor (
    id integer NOT NULL,
    perfil_id integer NOT NULL,
    proveedor_id integer NOT NULL,
    activo boolean DEFAULT true NOT NULL,
    fecha_creacion timestamp without time zone DEFAULT now() NOT NULL,
    fecha_modificacion timestamp without time zone,
    usuario_modificacion_id integer
);


ALTER TABLE proveedores.perfil_proveedor OWNER TO postgres;

--
-- Name: COLUMN perfil_proveedor.usuario_modificacion_id; Type: COMMENT; Schema: proveedores; Owner: postgres
--

COMMENT ON COLUMN proveedores.perfil_proveedor.usuario_modificacion_id IS 'Posible referencia externa a Terceros CRUD';


--
-- Name: perfil_proveedor_id_seq; Type: SEQUENCE; Schema: proveedores; Owner: postgres
--

CREATE SEQUENCE proveedores.perfil_proveedor_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE proveedores.perfil_proveedor_id_seq OWNER TO postgres;

--
-- Name: perfil_proveedor_id_seq; Type: SEQUENCE OWNED BY; Schema: proveedores; Owner: postgres
--

ALTER SEQUENCE proveedores.perfil_proveedor_id_seq OWNED BY proveedores.perfil_proveedor.id;


--
-- Name: persona_juridica_tipo_entidad; Type: TABLE; Schema: proveedores; Owner: postgres
--

CREATE TABLE proveedores.persona_juridica_tipo_entidad (
    id integer NOT NULL,
    nombre_perfil_entidad character varying(50),
    tipo_entidad_id integer NOT NULL,
    persona_juridica_id integer NOT NULL,
    activo boolean DEFAULT true NOT NULL,
    fecha_creacion timestamp without time zone DEFAULT now() NOT NULL,
    fecha_modificacion timestamp without time zone,
    usuario_modificacion_id integer
);


ALTER TABLE proveedores.persona_juridica_tipo_entidad OWNER TO postgres;

--
-- Name: COLUMN persona_juridica_tipo_entidad.usuario_modificacion_id; Type: COMMENT; Schema: proveedores; Owner: postgres
--

COMMENT ON COLUMN proveedores.persona_juridica_tipo_entidad.usuario_modificacion_id IS 'Posible referencia externa a Terceros CRUD';


--
-- Name: persona_juridica_tipo_entidad_id_seq; Type: SEQUENCE; Schema: proveedores; Owner: postgres
--

CREATE SEQUENCE proveedores.persona_juridica_tipo_entidad_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE proveedores.persona_juridica_tipo_entidad_id_seq OWNER TO postgres;

--
-- Name: persona_juridica_tipo_entidad_id_seq; Type: SEQUENCE OWNED BY; Schema: proveedores; Owner: postgres
--

ALTER SEQUENCE proveedores.persona_juridica_tipo_entidad_id_seq OWNED BY proveedores.persona_juridica_tipo_entidad.id;


--
-- Name: proveedor; Type: TABLE; Schema: proveedores; Owner: postgres
--

CREATE TABLE proveedores.proveedor (
    id integer NOT NULL,
    tipo_proveedor integer NOT NULL,
    descripcion_portafolio text,
    consentimiento boolean DEFAULT false,
    tercero_id integer,
    activo boolean DEFAULT true NOT NULL,
    fecha_creacion timestamp without time zone DEFAULT now() NOT NULL,
    fecha_modificacion timestamp without time zone,
    usuario_modificacion_id integer
);


ALTER TABLE proveedores.proveedor OWNER TO postgres;

--
-- Name: COLUMN proveedor.usuario_modificacion_id; Type: COMMENT; Schema: proveedores; Owner: postgres
--

COMMENT ON COLUMN proveedores.proveedor.usuario_modificacion_id IS 'Posible referencia externa a Terceros CRUD';


--
-- Name: proveedor_id_seq; Type: SEQUENCE; Schema: proveedores; Owner: postgres
--

CREATE SEQUENCE proveedores.proveedor_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE proveedores.proveedor_id_seq OWNER TO postgres;

--
-- Name: proveedor_id_seq; Type: SEQUENCE OWNED BY; Schema: proveedores; Owner: postgres
--

ALTER SEQUENCE proveedores.proveedor_id_seq OWNED BY proveedores.proveedor.id;


--
-- Name: proveedor_juridico; Type: TABLE; Schema: proveedores; Owner: postgres
--

CREATE TABLE proveedores.proveedor_juridico (
    id integer NOT NULL,
    procedencia_empresa_id integer NOT NULL,
    ciudad_origen_id integer,
    pais_origen_id integer,
    codigo_postal character varying(16),
    tipo_conformacion_id integer NOT NULL,
    exclusividad_producto boolean NOT NULL,
    regimen_contributivo integer NOT NULL,
    pyme boolean NOT NULL,
    registro_mercantil boolean NOT NULL,
    sujeto_retencion boolean NOT NULL,
    agente_retenedor boolean NOT NULL,
    responsable_ica boolean NOT NULL,
    responsable_iva boolean NOT NULL,
    activo boolean DEFAULT true NOT NULL,
    fecha_creacion timestamp without time zone DEFAULT now() NOT NULL,
    fecha_modificacion timestamp without time zone,
    usuario_modificacion_id integer
);


ALTER TABLE proveedores.proveedor_juridico OWNER TO postgres;

--
-- Name: COLUMN proveedor_juridico.usuario_modificacion_id; Type: COMMENT; Schema: proveedores; Owner: postgres
--

COMMENT ON COLUMN proveedores.proveedor_juridico.usuario_modificacion_id IS 'Posible referencia externa a Terceros CRUD';


--
-- Name: proveedor_juridico_id_seq; Type: SEQUENCE; Schema: proveedores; Owner: postgres
--

CREATE SEQUENCE proveedores.proveedor_juridico_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE proveedores.proveedor_juridico_id_seq OWNER TO postgres;

--
-- Name: proveedor_juridico_id_seq; Type: SEQUENCE OWNED BY; Schema: proveedores; Owner: postgres
--

ALTER SEQUENCE proveedores.proveedor_juridico_id_seq OWNED BY proveedores.proveedor_juridico.id;


--
-- Name: proveedor_natural; Type: TABLE; Schema: proveedores; Owner: postgres
--

CREATE TABLE proveedores.proveedor_natural (
    id integer NOT NULL,
    pais_nacimiento_id integer NOT NULL,
    cargo character varying(150),
    medicina_prepagada boolean,
    valor_uvt_prepagada numeric(20,7),
    interes_vivienda_afc numeric(5,4),
    dependiente_hijo_mejor_edad boolean,
    dependiente_hijo_menos23_estudiando boolean,
    dependiente_hijo_mas23_discapacitado boolean,
    dependiente_conyuge boolean,
    dependiente_padre_o_hermano boolean,
    pensionado boolean,
    valor_afc numeric(20,7),
    responsable_iva boolean,
    reside_colombia boolean,
    pension_voluntaria boolean,
    valor_pension_voluntaria numeric(20,7),
    experiencia_laboral integer,
    experiencia_profesional integer,
    activo boolean DEFAULT true NOT NULL,
    fecha_creacion timestamp without time zone DEFAULT now() NOT NULL,
    fecha_modificacion timestamp without time zone,
    usuario_modificacion_id integer
);


ALTER TABLE proveedores.proveedor_natural OWNER TO postgres;

--
-- Name: COLUMN proveedor_natural.usuario_modificacion_id; Type: COMMENT; Schema: proveedores; Owner: postgres
--

COMMENT ON COLUMN proveedores.proveedor_natural.usuario_modificacion_id IS 'Posible referencia externa a Terceros CRUD';


--
-- Name: proveedor_natural_id_seq; Type: SEQUENCE; Schema: proveedores; Owner: postgres
--

CREATE SEQUENCE proveedores.proveedor_natural_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE proveedores.proveedor_natural_id_seq OWNER TO postgres;

--
-- Name: proveedor_natural_id_seq; Type: SEQUENCE OWNED BY; Schema: proveedores; Owner: postgres
--

ALTER SEQUENCE proveedores.proveedor_natural_id_seq OWNED BY proveedores.proveedor_natural.id;


--
-- Name: representacion; Type: TABLE; Schema: proveedores; Owner: postgres
--

CREATE TABLE proveedores.representacion (
    id integer NOT NULL,
    proveedor_id integer NOT NULL,
    representante_id integer NOT NULL,
    activo boolean DEFAULT true NOT NULL,
    fecha_creacion timestamp without time zone DEFAULT now() NOT NULL,
    fecha_modificacion timestamp without time zone,
    usuario_modificacion_id integer
);


ALTER TABLE proveedores.representacion OWNER TO postgres;

--
-- Name: COLUMN representacion.usuario_modificacion_id; Type: COMMENT; Schema: proveedores; Owner: postgres
--

COMMENT ON COLUMN proveedores.representacion.usuario_modificacion_id IS 'Posible referencia externa a Terceros CRUD';


--
-- Name: representacion_id_seq; Type: SEQUENCE; Schema: proveedores; Owner: postgres
--

CREATE SEQUENCE proveedores.representacion_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE proveedores.representacion_id_seq OWNER TO postgres;

--
-- Name: representacion_id_seq; Type: SEQUENCE OWNED BY; Schema: proveedores; Owner: postgres
--

ALTER SEQUENCE proveedores.representacion_id_seq OWNED BY proveedores.representacion.id;


--
-- Name: titulo_proveedor_natural; Type: TABLE; Schema: proveedores; Owner: postgres
--

CREATE TABLE proveedores.titulo_proveedor_natural (
    id integer NOT NULL,
    titulo_obtenido character varying(150) NOT NULL,
    nucleo_basico_conocimiento_snies_id integer CONSTRAINT titulo_proveedor_natural_nucleo_basico_conocimiento_sn_not_null NOT NULL,
    proveedor_id integer NOT NULL,
    perfil_id integer NOT NULL,
    activo boolean DEFAULT true NOT NULL,
    fecha_creacion timestamp without time zone DEFAULT now() NOT NULL,
    fecha_modificacion timestamp without time zone,
    usuario_modificacion_id integer
);


ALTER TABLE proveedores.titulo_proveedor_natural OWNER TO postgres;

--
-- Name: COLUMN titulo_proveedor_natural.usuario_modificacion_id; Type: COMMENT; Schema: proveedores; Owner: postgres
--

COMMENT ON COLUMN proveedores.titulo_proveedor_natural.usuario_modificacion_id IS 'Posible referencia externa a Terceros CRUD';


--
-- Name: titulo_proveedor_natural_id_seq; Type: SEQUENCE; Schema: proveedores; Owner: postgres
--

CREATE SEQUENCE proveedores.titulo_proveedor_natural_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE proveedores.titulo_proveedor_natural_id_seq OWNER TO postgres;

--
-- Name: titulo_proveedor_natural_id_seq; Type: SEQUENCE OWNED BY; Schema: proveedores; Owner: postgres
--

ALTER SEQUENCE proveedores.titulo_proveedor_natural_id_seq OWNED BY proveedores.titulo_proveedor_natural.id;


--
-- Name: actividad_economica_proveedor id; Type: DEFAULT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.actividad_economica_proveedor ALTER COLUMN id SET DEFAULT nextval('proveedores.actividad_economica_proveedor_id_seq'::regclass);


--
-- Name: contacto id; Type: DEFAULT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.contacto ALTER COLUMN id SET DEFAULT nextval('proveedores.contacto_id_seq'::regclass);


--
-- Name: cuenta_bancaria id; Type: DEFAULT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.cuenta_bancaria ALTER COLUMN id SET DEFAULT nextval('proveedores.cuenta_bancaria_id_seq'::regclass);


--
-- Name: documento id; Type: DEFAULT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.documento ALTER COLUMN id SET DEFAULT nextval('proveedores.documento_id_seq'::regclass);


--
-- Name: estado_proveedor id; Type: DEFAULT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.estado_proveedor ALTER COLUMN id SET DEFAULT nextval('proveedores.estado_proveedor_id_seq'::regclass);


--
-- Name: perfil_proveedor id; Type: DEFAULT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.perfil_proveedor ALTER COLUMN id SET DEFAULT nextval('proveedores.perfil_proveedor_id_seq'::regclass);


--
-- Name: persona_juridica_tipo_entidad id; Type: DEFAULT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.persona_juridica_tipo_entidad ALTER COLUMN id SET DEFAULT nextval('proveedores.persona_juridica_tipo_entidad_id_seq'::regclass);


--
-- Name: proveedor id; Type: DEFAULT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.proveedor ALTER COLUMN id SET DEFAULT nextval('proveedores.proveedor_id_seq'::regclass);


--
-- Name: proveedor_juridico id; Type: DEFAULT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.proveedor_juridico ALTER COLUMN id SET DEFAULT nextval('proveedores.proveedor_juridico_id_seq'::regclass);


--
-- Name: proveedor_natural id; Type: DEFAULT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.proveedor_natural ALTER COLUMN id SET DEFAULT nextval('proveedores.proveedor_natural_id_seq'::regclass);


--
-- Name: representacion id; Type: DEFAULT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.representacion ALTER COLUMN id SET DEFAULT nextval('proveedores.representacion_id_seq'::regclass);


--
-- Name: titulo_proveedor_natural id; Type: DEFAULT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.titulo_proveedor_natural ALTER COLUMN id SET DEFAULT nextval('proveedores.titulo_proveedor_natural_id_seq'::regclass);


--
-- Data for Name: actividad_economica_proveedor; Type: TABLE DATA; Schema: proveedores; Owner: postgres
--

COPY proveedores.actividad_economica_proveedor (id, proveedor_id, actividad_economica_id, activo, fecha_creacion, fecha_modificacion, usuario_modificacion_id) FROM stdin;
\.


--
-- Data for Name: contacto; Type: TABLE DATA; Schema: proveedores; Owner: postgres
--

COPY proveedores.contacto (id, nombre_contacto, valor_especifico, proveedor_id, tipo_canal_contacto_id, proposito, extension, activo, fecha_creacion, fecha_modificacion, usuario_modificacion_id) FROM stdin;
1	Ana Perez	ana.perez@example.com	1	1	Notificaciones	\N	t	2026-11-13 22:34:48.274409	\N	\N
\.


--
-- Data for Name: cuenta_bancaria; Type: TABLE DATA; Schema: proveedores; Owner: postgres
--

COPY proveedores.cuenta_bancaria (id, tipo_cuenta_id, numero_cuenta, entidad_bancaria_id, contacto_id, monto_capital_autorizado, proveedor_id, activo, fecha_creacion, fecha_modificacion, usuario_modificacion_id) FROM stdin;
1	1	123456789	7	1	\N	1	t	2026-11-13 22:34:48.274409	\N	\N
\.


--
-- Data for Name: documento; Type: TABLE DATA; Schema: proveedores; Owner: postgres
--

COPY proveedores.documento (id, tipo_documento_id, enlace, proveedor_id, activo, fecha_creacion, fecha_modificacion, usuario_modificacion_id) FROM stdin;
\.


--
-- Data for Name: estado_proveedor; Type: TABLE DATA; Schema: proveedores; Owner: postgres
--

COPY proveedores.estado_proveedor (id, estado_id, proveedor_id, observacion, activo, fecha_creacion, fecha_modificacion, usuario_modificacion_id) FROM stdin;
\.


--
-- Data for Name: perfil_proveedor; Type: TABLE DATA; Schema: proveedores; Owner: postgres
--

COPY proveedores.perfil_proveedor (id, perfil_id, proveedor_id, activo, fecha_creacion, fecha_modificacion, usuario_modificacion_id) FROM stdin;
\.


--
-- Data for Name: persona_juridica_tipo_entidad; Type: TABLE DATA; Schema: proveedores; Owner: postgres
--

COPY proveedores.persona_juridica_tipo_entidad (id, nombre_perfil_entidad, tipo_entidad_id, persona_juridica_id, activo, fecha_creacion, fecha_modificacion, usuario_modificacion_id) FROM stdin;
\.


--
-- Data for Name: proveedor; Type: TABLE DATA; Schema: proveedores; Owner: postgres
--

COPY proveedores.proveedor (id, tipo_proveedor, descripcion_portafolio, consentimiento, tercero_id, activo, fecha_creacion, fecha_modificacion, usuario_modificacion_id) FROM stdin;
1	1	Servicios de consultoria	t	101	t	2026-11-13 22:34:48.274409	\N	\N
2	2	Suministro de papeleria	f	102	t	2026-11-13 22:34:48.274409	\N	\N
\.


--
-- Data for Name: proveedor_juridico; Type: TABLE DATA; Schema: proveedores; Owner: postgres
--

COPY proveedores.proveedor_juridico (id, procedencia_empresa_id, ciudad_origen_id, pais_origen_id, codigo_postal, tipo_conformacion_id, exclusividad_producto, regimen_contributivo, pyme, registro_mercantil, sujeto_retencion, agente_retenedor, responsable_ica, responsable_iva, activo, fecha_creacion, fecha_modificacion, usuario_modificacion_id) FROM stdin;
2	1	\N	\N	110111	1	f	1	t	t	t	f	t	t	t	2026-11-13 22:34:48.274409	\N	\N
\.


--
-- Data for Name: proveedor_natural; Type: TABLE DATA; Schema: proveedores; Owner: postgres
--

COPY proveedores.proveedor_natural (id, pais_nacimiento_id, cargo, medicina_prepagada, valor_uvt_prepagada, interes_vivienda_afc, dependiente_hijo_mejor_edad, dependiente_hijo_menos23_estudiando, dependiente_hijo_mas23_discapacitado, dependiente_conyuge, dependiente_padre_o_hermano, pensionado, valor_afc, responsable_iva, reside_colombia, pension_voluntaria, valor_pension_voluntaria, experiencia_laboral, experiencia_profesional, activo, fecha_creacion, fecha_modificacion, usuario_modificacion_id) FROM stdin;
1	1	Consultora	\N	\N	\N	\N	\N	\N	\N	\N	\N	\N	\N	\N	\N	\N	24	60	t	2026-11-13 22:34:48.274409	\N	\N
\.


--
-- Data for Name: representacion; Type: TABLE DATA; Schema: proveedores; Owner: postgres
--

COPY proveedores.representacion (id, proveedor_id, representante_id, activo, fecha_creacion, fecha_modificacion, usuario_modificacion_id) FROM stdin;
\.


--
-- Data for Name: titulo_proveedor_natural; Type: TABLE DATA; Schema: proveedores; Owner: postgres
--

COPY proveedores.titulo_proveedor_natural (id, titulo_obtenido, nucleo_basico_conocimiento_snies_id, proveedor_id, perfil_id, activo, fecha_creacion, fecha_modificacion, usuario_modificacion_id) FROM stdin;
\.


--
-- Name: actividad_economica_proveedor_id_seq; Type: SEQUENCE SET; Schema: proveedores; Owner: postgres
--

SELECT pg_catalog.setval('proveedores.actividad_economica_proveedor_id_seq', 1, false);


--
-- Name: contacto_id_seq; Type: SEQUENCE SET; Schema: proveedores; Owner: postgres
--

SELECT pg_catalog.setval('proveedores.contacto_id_seq', 1, true);


--
-- Name: cuenta_bancaria_id_seq; Type: SEQUENCE SET; Schema: proveedores; Owner: postgres
--

SELECT pg_catalog.setval('proveedores.cuenta_bancaria_id_seq', 1, true);


--
-- Name: documento_id_seq; Type: SEQUENCE SET; Schema: proveedores; Owner: postgres
--

SELECT pg_catalog.setval('proveedores.documento_id_seq', 1, false);


--
-- Name: estado_proveedor_id_seq; Type: SEQUENCE SET; Schema: proveedores; Owner: postgres
--

SELECT pg_catalog.setval('proveedores.estado_proveedor_id_seq', 1, false);


--
-- Name: perfil_proveedor_id_seq; Type: SEQUENCE SET; Schema: proveedores; Owner: postgres
--

SELECT pg_catalog.setval('proveedores.perfil_proveedor_id_seq', 1, false);


--
-- Name: persona_juridica_tipo_entidad_id_seq; Type: SEQUENCE SET; Schema: proveedores; Owner: postgres
--

SELECT pg_catalog.setval('proveedores.persona_juridica_tipo_entidad_id_seq', 1, false);


--
-- Name: proveedor_id_seq; Type: SEQUENCE SET; Schema: proveedores; Owner: postgres
--

SELECT pg_catalog.setval('proveedores.proveedor_id_seq', 2, true);


--
-- Name: proveedor_juridico_id_seq; Type: SEQUENCE SET; Schema: proveedores; Owner: postgres
--

SELECT pg_catalog.setval('proveedores.proveedor_juridico_id_seq', 2, true);


--
-- Name: proveedor_natural_id_seq; Type: SEQUENCE SET; Schema: proveedores; Owner: postgres
--

SELECT pg_catalog.setval('proveedores.proveedor_natural_id_seq', 1, true);


--
-- Name: representacion_id_seq; Type: SEQUENCE SET; Schema: proveedores; Owner: postgres
--

SELECT pg_catalog.setval('proveedores.representacion_id_seq', 1, false);


--
-- Name: titulo_proveedor_natural_id_seq; Type: SEQUENCE SET; Schema: proveedores; Owner: postgres
--

SELECT pg_catalog.setval('proveedores.titulo_proveedor_natural_id_seq', 1, false);


--
-- Name: actividad_economica_proveedor actividad_economica_proveedor_pkey; Type: CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.actividad_economica_proveedor
    ADD CONSTRAINT actividad_economica_proveedor_pkey PRIMARY KEY (id);


--
-- Name: contacto contacto_pkey; Type: CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.contacto
    ADD CONSTRAINT contacto_pkey PRIMARY KEY (id);


--
-- Name: cuenta_bancaria cuenta_bancaria_pkey; Type: CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.cuenta_bancaria
    ADD CONSTRAINT cuenta_bancaria_pkey PRIMARY KEY (id);


--
-- Name: documento documento_pkey; Type: CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.documento
    ADD CONSTRAINT documento_pkey PRIMARY KEY (id);


--
-- Name: estado_proveedor estado_proveedor_pkey; Type: CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.estado_proveedor
    ADD CONSTRAINT estado_proveedor_pkey PRIMARY KEY (id);


--
-- Name: perfil_proveedor perfil_proveedor_pkey; Type: CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.perfil_proveedor
    ADD CONSTRAINT perfil_proveedor_pkey PRIMARY KEY (id);


--
-- Name: persona_juridica_tipo_entidad persona_juridica_tipo_entidad_pkey; Type: CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.persona_juridica_tipo_entidad
    ADD CONSTRAINT persona_juridica_tipo_entidad_pkey PRIMARY KEY (id);


--
-- Name: proveedor_juridico proveedor_juridico_pkey; Type: CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.proveedor_juridico
    ADD CONSTRAINT proveedor_juridico_pkey PRIMARY KEY (id);


--
-- Name: proveedor_natural proveedor_natural_pkey; Type: CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.proveedor_natural
    ADD CONSTRAINT proveedor_natural_pkey PRIMARY KEY (id);


--
-- Name: proveedor proveedor_pkey; Type: CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.proveedor
    ADD CONSTRAINT proveedor_pkey PRIMARY KEY (id);


--
-- Name: representacion representacion_pkey; Type: CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.representacion
    ADD CONSTRAINT representacion_pkey PRIMARY KEY (id);


--
-- Name: titulo_proveedor_natural titulo_proveedor_natural_pkey; Type: CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.titulo_proveedor_natural
    ADD CONSTRAINT titulo_proveedor_natural_pkey PRIMARY KEY (id);


--
-- Name: idx_actividad_economica_proveedor_actividad_economica_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_actividad_economica_proveedor_actividad_economica_id ON proveedores.actividad_economica_proveedor USING btree (actividad_economica_id);


--
-- Name: idx_actividad_economica_proveedor_activo; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_actividad_economica_proveedor_activo ON proveedores.actividad_economica_proveedor USING btree (activo);


--
-- Name: idx_actividad_economica_proveedor_fecha_creacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_actividad_economica_proveedor_fecha_creacion ON proveedores.actividad_economica_proveedor USING btree (fecha_creacion);


--
-- Name: idx_actividad_economica_proveedor_fecha_modificacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_actividad_economica_proveedor_fecha_modificacion ON proveedores.actividad_economica_proveedor USING btree (fecha_modificacion);


--
-- Name: idx_actividad_economica_proveedor_proveedor_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_actividad_economica_proveedor_proveedor_id ON proveedores.actividad_economica_proveedor USING btree (proveedor_id);


--
-- Name: idx_actividad_economica_proveedor_usuario_modificacion_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_actividad_economica_proveedor_usuario_modificacion_id ON proveedores.actividad_economica_proveedor USING btree (usuario_modificacion_id);


--
-- Name: idx_contacto_activo; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_contacto_activo ON proveedores.contacto USING btree (activo);


--
-- Name: idx_contacto_fecha_creacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_contacto_fecha_creacion ON proveedores.contacto USING btree (fecha_creacion);


--
-- Name: idx_contacto_fecha_modificacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_contacto_fecha_modificacion ON proveedores.contacto USING btree (fecha_modificacion);


--
-- Name: idx_contacto_proveedor_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_contacto_proveedor_id ON proveedores.contacto USING btree (proveedor_id);


--
-- Name: idx_contacto_tipo_canal_contacto_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_contacto_tipo_canal_contacto_id ON proveedores.contacto USING btree (tipo_canal_contacto_id);


--
-- Name: idx_contacto_usuario_modificacion_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_contacto_usuario_modificacion_id ON proveedores.contacto USING btree (usuario_modificacion_id);


--
-- Name: idx_cuenta_bancaria_activo; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_cuenta_bancaria_activo ON proveedores.cuenta_bancaria USING btree (activo);


--
-- Name: idx_cuenta_bancaria_contacto_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_cuenta_bancaria_contacto_id ON proveedores.cuenta_bancaria USING btree (contacto_id);


--
-- Name: idx_cuenta_bancaria_entidad_bancaria_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_cuenta_bancaria_entidad_bancaria_id ON proveedores.cuenta_bancaria USING btree (entidad_bancaria_id);


--
-- Name: idx_cuenta_bancaria_fecha_creacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_cuenta_bancaria_fecha_creacion ON proveedores.cuenta_bancaria USING btree (fecha_creacion);


--
-- Name: idx_cuenta_bancaria_fecha_modificacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_cuenta_bancaria_fecha_modificacion ON proveedores.cuenta_bancaria USING btree (fecha_modificacion);


--
-- Name: idx_cuenta_bancaria_proveedor_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_cuenta_bancaria_proveedor_id ON proveedores.cuenta_bancaria USING btree (proveedor_id);


--
-- Name: idx_cuenta_bancaria_tipo_cuenta_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_cuenta_bancaria_tipo_cuenta_id ON proveedores.cuenta_bancaria USING btree (tipo_cuenta_id);


--
-- Name: idx_cuenta_bancaria_usuario_modificacion_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_cuenta_bancaria_usuario_modificacion_id ON proveedores.cuenta_bancaria USING btree (usuario_modificacion_id);


--
-- Name: idx_documento_activo; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_documento_activo ON proveedores.documento USING btree (activo);


--
-- Name: idx_documento_fecha_creacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_documento_fecha_creacion ON proveedores.documento USING btree (fecha_creacion);


--
-- Name: idx_documento_fecha_modificacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_documento_fecha_modificacion ON proveedores.documento USING btree (fecha_modificacion);


--
-- Name: idx_documento_proveedor_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_documento_proveedor_id ON proveedores.documento USING btree (proveedor_id);


--
-- Name: idx_documento_tipo_documento_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_documento_tipo_documento_id ON proveedores.documento USING btree (tipo_documento_id);


--
-- Name: idx_documento_usuario_modificacion_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_documento_usuario_modificacion_id ON proveedores.documento USING btree (usuario_modificacion_id);


--
-- Name: idx_estado_proveedor_activo; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_estado_proveedor_activo ON proveedores.estado_proveedor USING btree (activo);


--
-- Name: idx_estado_proveedor_estado_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_estado_proveedor_estado_id ON proveedores.estado_proveedor USING btree (estado_id);


--
-- Name: idx_estado_proveedor_fecha_creacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_estado_proveedor_fecha_creacion ON proveedores.estado_proveedor USING btree (fecha_creacion);


--
-- Name: idx_estado_proveedor_fecha_modificacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_estado_proveedor_fecha_modificacion ON proveedores.estado_proveedor USING btree (fecha_modificacion);


--
-- Name: idx_estado_proveedor_proveedor_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_estado_proveedor_proveedor_id ON proveedores.estado_proveedor USING btree (proveedor_id);


--
-- Name: idx_estado_proveedor_usuario_modificacion_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_estado_proveedor_usuario_modificacion_id ON proveedores.estado_proveedor USING btree (usuario_modificacion_id);


--
-- Name: idx_perfil_proveedor_activo; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_perfil_proveedor_activo ON proveedores.perfil_proveedor USING btree (activo);


--
-- Name: idx_perfil_proveedor_fecha_creacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_perfil_proveedor_fecha_creacion ON proveedores.perfil_proveedor USING btree (fecha_creacion);


--
-- Name: idx_perfil_proveedor_fecha_modificacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_perfil_proveedor_fecha_modificacion ON proveedores.perfil_proveedor USING btree (fecha_modificacion);


--
-- Name: idx_perfil_proveedor_perfil_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_perfil_proveedor_perfil_id ON proveedores.perfil_proveedor USING btree (perfil_id);


--
-- Name: idx_perfil_proveedor_proveedor_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_perfil_proveedor_proveedor_id ON proveedores.perfil_proveedor USING btree (proveedor_id);


--
-- Name: idx_perfil_proveedor_usuario_modificacion_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_perfil_proveedor_usuario_modificacion_id ON proveedores.perfil_proveedor USING btree (usuario_modificacion_id);


--
-- Name: idx_persona_juridica_tipo_entidad_activo; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_persona_juridica_tipo_entidad_activo ON proveedores.persona_juridica_tipo_entidad USING btree (activo);


--
-- Name: idx_persona_juridica_tipo_entidad_fecha_creacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_persona_juridica_tipo_entidad_fecha_creacion ON proveedores.persona_juridica_tipo_entidad USING btree (fecha_creacion);


--
-- Name: idx_persona_juridica_tipo_entidad_fecha_modificacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_persona_juridica_tipo_entidad_fecha_modificacion ON proveedores.persona_juridica_tipo_entidad USING btree (fecha_modificacion);


--
-- Name: idx_persona_juridica_tipo_entidad_persona_juridica_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_persona_juridica_tipo_entidad_persona_juridica_id ON proveedores.persona_juridica_tipo_entidad USING btree (persona_juridica_id);


--
-- Name: idx_persona_juridica_tipo_entidad_tipo_entidad_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_persona_juridica_tipo_entidad_tipo_entidad_id ON proveedores.persona_juridica_tipo_entidad USING btree (tipo_entidad_id);


--
-- Name: idx_persona_juridica_tipo_entidad_usuario_modificacion_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_persona_juridica_tipo_entidad_usuario_modificacion_id ON proveedores.persona_juridica_tipo_entidad USING btree (usuario_modificacion_id);


--
-- Name: idx_proveedor_activo; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_proveedor_activo ON proveedores.proveedor USING btree (activo);


--
-- Name: idx_proveedor_fecha_creacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_proveedor_fecha_creacion ON proveedores.proveedor USING btree (fecha_creacion);


--
-- Name: idx_proveedor_fecha_modificacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_proveedor_fecha_modificacion ON proveedores.proveedor USING btree (fecha_modificacion);


--
-- Name: idx_proveedor_juridico_activo; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_proveedor_juridico_activo ON proveedores.proveedor_juridico USING btree (activo);


--
-- Name: idx_proveedor_juridico_ciudad_origen_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_proveedor_juridico_ciudad_origen_id ON proveedores.proveedor_juridico USING btree (ciudad_origen_id);


--
-- Name: idx_proveedor_juridico_fecha_creacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_proveedor_juridico_fecha_creacion ON proveedores.proveedor_juridico USING btree (fecha_creacion);


--
-- Name: idx_proveedor_juridico_fecha_modificacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_proveedor_juridico_fecha_modificacion ON proveedores.proveedor_juridico USING btree (fecha_modificacion);


--
-- Name: idx_proveedor_juridico_pais_origen_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_proveedor_juridico_pais_origen_id ON proveedores.proveedor_juridico USING btree (pais_origen_id);


--
-- Name: idx_proveedor_juridico_procedencia_empresa_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_proveedor_juridico_procedencia_empresa_id ON proveedores.proveedor_juridico USING btree (procedencia_empresa_id);


--
-- Name: idx_proveedor_juridico_regimen_contributivo; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_proveedor_juridico_regimen_contributivo ON proveedores.proveedor_juridico USING btree (regimen_contributivo);


--
-- Name: idx_proveedor_juridico_tipo_conformacion_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_proveedor_juridico_tipo_conformacion_id ON proveedores.proveedor_juridico USING btree (tipo_conformacion_id);


--
-- Name: idx_proveedor_juridico_usuario_modificacion_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_proveedor_juridico_usuario_modificacion_id ON proveedores.proveedor_juridico USING btree (usuario_modificacion_id);


--
-- Name: idx_proveedor_natural_activo; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_proveedor_natural_activo ON proveedores.proveedor_natural USING btree (activo);


--
-- Name: idx_proveedor_natural_fecha_creacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_proveedor_natural_fecha_creacion ON proveedores.proveedor_natural USING btree (fecha_creacion);


--
-- Name: idx_proveedor_natural_fecha_modificacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_proveedor_natural_fecha_modificacion ON proveedores.proveedor_natural USING btree (fecha_modificacion);


--
-- Name: idx_proveedor_natural_pais_nacimiento_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_proveedor_natural_pais_nacimiento_id ON proveedores.proveedor_natural USING btree (pais_nacimiento_id);


--
-- Name: idx_proveedor_natural_usuario_modificacion_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_proveedor_natural_usuario_modificacion_id ON proveedores.proveedor_natural USING btree (usuario_modificacion_id);


--
-- Name: idx_proveedor_tercero_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_proveedor_tercero_id ON proveedores.proveedor USING btree (tercero_id);


--
-- Name: idx_proveedor_tipo_proveedor; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_proveedor_tipo_proveedor ON proveedores.proveedor USING btree (tipo_proveedor);


--
-- Name: idx_proveedor_usuario_modificacion_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_proveedor_usuario_modificacion_id ON proveedores.proveedor USING btree (usuario_modificacion_id);


--
-- Name: idx_representacion_activo; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_representacion_activo ON proveedores.representacion USING btree (activo);


--
-- Name: idx_representacion_fecha_creacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_representacion_fecha_creacion ON proveedores.representacion USING btree (fecha_creacion);


--
-- Name: idx_representacion_fecha_modificacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_representacion_fecha_modificacion ON proveedores.representacion USING btree (fecha_modificacion);


--
-- Name: idx_representacion_proveedor_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_representacion_proveedor_id ON proveedores.representacion USING btree (proveedor_id);


--
-- Name: idx_representacion_representante_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_representacion_representante_id ON proveedores.representacion USING btree (representante_id);


--
-- Name: idx_representacion_usuario_modificacion_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_representacion_usuario_modificacion_id ON proveedores.representacion USING btree (usuario_modificacion_id);


--
-- Name: idx_titulo_proveedor_natural_activo; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_titulo_proveedor_natural_activo ON proveedores.titulo_proveedor_natural USING btree (activo);


--
-- Name: idx_titulo_proveedor_natural_fecha_creacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_titulo_proveedor_natural_fecha_creacion ON proveedores.titulo_proveedor_natural USING btree (fecha_creacion);


--
-- Name: idx_titulo_proveedor_natural_fecha_modificacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_titulo_proveedor_natural_fecha_modificacion ON proveedores.titulo_proveedor_natural USING btree (fecha_modificacion);


--
-- Name: idx_titulo_proveedor_natural_nucleo_basico_conocimiento_snies_i; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_titulo_proveedor_natural_nucleo_basico_conocimiento_snies_i ON proveedores.titulo_proveedor_natural USING btree (nucleo_basico_conocimiento_snies_id);


--
-- Name: idx_titulo_proveedor_natural_perfil_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_titulo_proveedor_natural_perfil_id ON proveedores.titulo_proveedor_natural USING btree (perfil_id);


--
-- Name: idx_titulo_proveedor_natural_proveedor_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_titulo_proveedor_natural_proveedor_id ON proveedores.titulo_proveedor_natural USING btree (proveedor_id);


--
-- Name: idx_titulo_proveedor_natural_usuario_modificacion_id; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX idx_titulo_proveedor_natural_usuario_modificacion_id ON proveedores.titulo_proveedor_natural USING btree (usuario_modificacion_id);


--
-- Name: pk_actividad_economica_proveedor; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX pk_actividad_economica_proveedor ON proveedores.actividad_economica_proveedor USING btree (id);


--
-- Name: pk_contacto; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX pk_contacto ON proveedores.contacto USING btree (id);


--
-- Name: pk_cuenta_bancaria; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX pk_cuenta_bancaria ON proveedores.cuenta_bancaria USING btree (id);


--
-- Name: pk_documento; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX pk_documento ON proveedores.documento USING btree (id);


--
-- Name: pk_estado_proveedor; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX pk_estado_proveedor ON proveedores.estado_proveedor USING btree (id);


--
-- Name: pk_perfil_proveedor; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX pk_perfil_proveedor ON proveedores.perfil_proveedor USING btree (id);


--
-- Name: pk_persona_juridica_tipo_entidad; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX pk_persona_juridica_tipo_entidad ON proveedores.persona_juridica_tipo_entidad USING btree (id);


--
-- Name: pk_proveedor; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX pk_proveedor ON proveedores.proveedor USING btree (id);


--
-- Name: pk_proveedor_juridico; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX pk_proveedor_juridico ON proveedores.proveedor_juridico USING btree (id);


--
-- Name: pk_proveedor_natural; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX pk_proveedor_natural ON proveedores.proveedor_natural USING btree (id);


--
-- Name: pk_representacion; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX pk_representacion ON proveedores.representacion USING btree (id);


--
-- Name: pk_titulo_proveedor_natural; Type: INDEX; Schema: proveedores; Owner: postgres
--

CREATE INDEX pk_titulo_proveedor_natural ON proveedores.titulo_proveedor_natural USING btree (id);


--
-- Name: actividad_economica_proveedor fk_actividad_economica_proveedor_proveedor; Type: FK CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.actividad_economica_proveedor
    ADD CONSTRAINT fk_actividad_economica_proveedor_proveedor FOREIGN KEY (proveedor_id) REFERENCES proveedores.proveedor(id) DEFERRABLE;


--
-- Name: contacto fk_contacto_proveedor; Type: FK CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.contacto
    ADD CONSTRAINT fk_contacto_proveedor FOREIGN KEY (proveedor_id) REFERENCES proveedores.proveedor(id) DEFERRABLE;


--
-- Name: cuenta_bancaria fk_cuenta_bancaria_contacto; Type: FK CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.cuenta_bancaria
    ADD CONSTRAINT fk_cuenta_bancaria_contacto FOREIGN KEY (contacto_id) REFERENCES proveedores.contacto(id) DEFERRABLE;


--
-- Name: cuenta_bancaria fk_cuenta_bancaria_proveedor; Type: FK CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.cuenta_bancaria
    ADD CONSTRAINT fk_cuenta_bancaria_proveedor FOREIGN KEY (proveedor_id) REFERENCES proveedores.proveedor(id) DEFERRABLE;


--
-- Name: documento fk_documento_proveedor; Type: FK CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.documento
    ADD CONSTRAINT fk_documento_proveedor FOREIGN KEY (proveedor_id) REFERENCES proveedores.proveedor(id) DEFERRABLE;


--
-- Name: estado_proveedor fk_estado_proveedor_proveedor; Type: FK CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.estado_proveedor
    ADD CONSTRAINT fk_estado_proveedor_proveedor FOREIGN KEY (proveedor_id) REFERENCES proveedores.proveedor(id) DEFERRABLE;


--
-- Name: perfil_proveedor fk_perfil_proveedor_proveedor; Type: FK CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.perfil_proveedor
    ADD CONSTRAINT fk_perfil_proveedor_proveedor FOREIGN KEY (proveedor_id) REFERENCES proveedores.proveedor(id) DEFERRABLE;


--
-- Name: persona_juridica_tipo_entidad fk_persona_juridica_tipo_entidad_proveedor_juridico; Type: FK CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.persona_juridica_tipo_entidad
    ADD CONSTRAINT fk_persona_juridica_tipo_entidad_proveedor_juridico FOREIGN KEY (persona_juridica_id) REFERENCES proveedores.proveedor_juridico(id) DEFERRABLE;


--
-- Name: proveedor_juridico fk_proveedor_juridico_proveedor; Type: FK CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.proveedor_juridico
    ADD CONSTRAINT fk_proveedor_juridico_proveedor FOREIGN KEY (id) REFERENCES proveedores.proveedor(id) DEFERRABLE;


--
-- Name: proveedor_natural fk_proveedor_natural_proveedor; Type: FK CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.proveedor_natural
    ADD CONSTRAINT fk_proveedor_natural_proveedor FOREIGN KEY (id) REFERENCES proveedores.proveedor(id) DEFERRABLE;


--
-- Name: representacion fk_representacion_proveedor; Type: FK CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.representacion
    ADD CONSTRAINT fk_representacion_proveedor FOREIGN KEY (proveedor_id) REFERENCES proveedores.proveedor(id) DEFERRABLE;


--
-- Name: titulo_proveedor_natural fk_titulo_proveedor_natural_proveedor; Type: FK CONSTRAINT; Schema: proveedores; Owner: postgres
--

ALTER TABLE ONLY proveedores.titulo_proveedor_natural
    ADD CONSTRAINT fk_titulo_proveedor_natural_proveedor FOREIGN KEY (proveedor_id) REFERENCES proveedores.proveedor(id) DEFERRABLE;


--
-- PostgreSQL database dump complete
--

\unrestrict e3ODu8pVl9WlvyYLVV1RavpJhJvVp6AZJWuibrkdp6S5zHw8oZ3dgZ8P7U0ltUd

