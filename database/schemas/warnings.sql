CREATE SCHEMA IF NOT EXISTS warnings;

CREATE TABLE IF NOT EXISTS warnings.warnings (
    id serial,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    issued timestamptz NOT NULL,
    starts timestamptz DEFAULT NULL,
    expires timestamptz NOT NULL,
    ends timestamptz DEFAULT NULL,
    end_initial timestamptz DEFAULT NULL,
    text text NOT NULL,
    wfo char(4) NOT NULL,
    action char(3) NOT NULL,
    class char(1) NOT NULL,
    phenomena char(2) NOT NULL,
    significance char(1) NOT NULL,
    event_number smallint NOT NULL,
    year smallint NOT NULL,
    title varchar(128) NOT NULL,
    is_emergency boolean DEFAULT false,
    is_pds boolean DEFAULT false,
    geom geometry(MultiPolygon, 4326),
    direction int,
    location geometry(MultiPoint, 4326),
    speed int,
    speed_text varchar(30),
    tml_time timestamptz,
    ugc char(6)[],
    tornado varchar(64),
    damage varchar(64),
    hail_threat varchar(64),
    hail_tag varchar(64),
    wind_threat varchar(64),
    wind_tag varchar(64),
    flash_flood varchar(64),
    rainfall_tag varchar(64),
    flood_tag_dam varchar(64),
    spout_tag varchar(64),
    snow_squall varchar(64),
    snow_squall_tag varchar(64),
	PRIMARY KEY (wfo, phenomena, significance, event_number, year, id)
);
CREATE INDEX IF NOT EXISTS warnings_issued ON warnings.warnings(issued);
CREATE INDEX IF NOT EXISTS warnings_starts ON warnings.warnings(starts);
CREATE INDEX IF NOT EXISTS warnings_expires ON warnings.warnings(expires);
CREATE INDEX IF NOT EXISTS warnings_ends ON warnings.warnings(ends);
CREATE INDEX IF NOT EXISTS warnings_phenomena_significance ON warnings.warnings(phenomena, significance);