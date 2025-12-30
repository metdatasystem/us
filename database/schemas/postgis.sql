CREATE EXTENSION postgis;

CREATE SCHEMA IF NOT EXISTS postgis;
ALTER SCHEMA postgis OWNER TO mds;

-- States --
CREATE TABLE IF NOT EXISTS postgis.states (
    id varchar(2) PRIMARY KEY,
    name varchar(50) NOT NULL,
    fips varchar(2) UNIQUE NOT NULL,
    is_offshore boolean NOT NULL
);
ALTER TABLE postgis.states OWNER TO mds;
GRANT ALL ON TABLE postgis.states TO postgis;
GRANT SELECT ON TABLE postgis.states TO nobody, api_service;

-- Office --
CREATE TABLE IF NOT EXISTS postgis.offices (
    id char(3) PRIMARY KEY,
    icao varchar(4) UNIQUE NOT NULL,
    name varchar(50) NOT NULL,
    state char(2) NOT NULL REFERENCES postgis.states(id),
    location geometry(Point, 4326)
);
ALTER TABLE postgis.offices OWNER TO mds;
GRANT ALL ON TABLE postgis.offices TO postgis;
GRANT SELECT ON TABLE postgis.offices TO nobody, api_service;

-- County Warning Area --
CREATE TABLE IF NOT EXISTS postgis.cwas (
    id char(3) PRIMARY KEY,
    name varchar(50) NOT NULL,
    area real NOT NULL,
    geom geometry(MultiPolygon, 4326) NOT NULL,
    wfo char(3) NOT NULL REFERENCES postgis.offices(id),
    region char(2) NOT NULL,
    valid_from timestamptz NOT NULL
);
ALTER TABLE postgis.cwas OWNER TO mds;
GRANT ALL ON TABLE postgis.cwas TO postgis;
GRANT SELECT ON TABLE postgis.cwas TO nobody, api_service;

-- UGC (Universal Geographic Code) --
CREATE TABLE IF NOT EXISTS postgis.ugcs (
    id serial UNIQUE PRIMARY KEY,
    ugc char(6) NOT NULL,
    name varchar(256) NOT NULL,
    state char(2) NOT NULL REFERENCES postgis.states(id),
    type char(1) NOT NULL,
    number smallint NOT NULL,
    area real NOT NULL,
    geom geometry(MultiPolygon, 4326) NOT NULL,
    cwa char(3)[] NOT NULL,
    is_marine boolean,
    is_fire boolean,
    valid_from timestamptz DEFAULT CURRENT_TIMESTAMP,
    valid_to timestamptz
);
CREATE INDEX IF NOT EXISTS ugc_ugc ON postgis.ugcs(ugc);
CREATE INDEX IF NOT EXISTS ugc_geom ON postgis.ugcs USING GIST(geom);
ALTER TABLE postgis.ugcs OWNER TO mds;
GRANT ALL ON TABLE postgis.ugcs TO postgis;
GRANT SELECT ON TABLE postgis.ugcs TO nobody, api_service;