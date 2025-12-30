CREATE SCHEMA IF NOT EXISTS vtec;
ALTER SCHEMA vtec OWNER TO mds;

-- Phenomena types
CREATE TABLE IF NOT EXISTS vtec.phenomena (
    id char(2) PRIMARY KEY,
    name varchar(64) NOT NULL,
    description varchar(64)
);
ALTER TABLE vtec.phenomena OWNER TO mds;
GRANT SELECT ON TABLE vtec.phenomena TO awips_service;
GRANT SELECT ON TABLE vtec.phenomena TO nobody, api_service;

-- Significance levels
CREATE TABLE IF NOT EXISTS vtec.significance (
    id char(1) PRIMARY KEY,
    name varchar(64) NOT NULL,
    description varchar(64)
);
ALTER TABLE vtec.significance OWNER TO mds;
GRANT SELECT ON TABLE vtec.significance TO awips_service;
GRANT SELECT ON TABLE vtec.significance TO nobody, api_service;

-- Action types
CREATE TABLE IF NOT EXISTS vtec.action (
    id char(3) PRIMARY KEY,
    name varchar(64) NOT NULL,
    description varchar(64)
);
ALTER TABLE vtec.action OWNER TO mds;
GRANT SELECT ON TABLE vtec.action TO awips_service;
GRANT SELECT ON TABLE vtec.action TO nobody, api_service;

-- VTEC Event --
CREATE TABLE IF NOT EXISTS vtec.events (
    id serial,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    issued timestamptz NOT NULL,
    starts timestamptz,
    expires timestamptz NOT NULL,
    ends timestamptz DEFAULT NULL,
    end_initial timestamptz DEFAULT NULL,
    class char(1) NOT NULL,
    phenomena char(2) NOT NULL REFERENCES vtec.phenomena(id),
    wfo char(4) NOT NULL REFERENCES postgis.offices(icao),
    significance char(1) NOT NULL REFERENCES vtec.significance(id),
    event_number smallint NOT NULL,
    year smallint NOT NULL,
    title varchar(128) NOT NULL,
    is_emergency boolean DEFAULT false,
    is_pds boolean DEFAULT false,
	PRIMARY KEY (wfo, phenomena, significance, event_number, year)
) PARTITION BY LIST (year);
ALTER TABLE vtec.events OWNER TO mds;
GRANT ALL ON TABLE vtec.events TO awips_service;
GRANT SELECT ON TABLE vtec.events TO nobody, api_service;

-- VTEC UGC --
CREATE TABLE IF NOT EXISTS vtec.ugcs (
    id serial,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    wfo char(4) NOT NULL,
    phenomena char(2) NOT NULL,
    significance char(1) NOT NULL,
    event_number smallint NOT NULL,
    ugc integer NOT NULL REFERENCES postgis.ugcs(id),
    issued timestamptz NOT NULL,
    starts timestamptz DEFAULT NULL,
    expires timestamptz NOT NULL,
    ends timestamptz DEFAULT NULL,
    end_initial timestamptz DEFAULT NULL,
    action char(3) NOT NULL REFERENCES vtec.action(id),
    year smallint NOT NULL,
	FOREIGN KEY (wfo, phenomena, significance, event_number, year) 
        REFERENCES vtec.events(wfo, phenomena, significance, event_number, year) ON DELETE CASCADE,
    PRIMARY KEY (wfo, phenomena, significance, event_number, year, ugc)
) PARTITION BY LIST (year);
ALTER TABLE vtec.ugcs OWNER TO mds;
GRANT ALL ON TABLE vtec.ugcs TO awips_service;
GRANT SELECT ON TABLE vtec.ugcs TO nobody, api_service;

-- VTEC Event Updates --
CREATE TABLE IF NOT EXISTS vtec.updates (
    id serial,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    issued timestamptz NOT NULL,
    starts timestamptz DEFAULT NULL,
    expires timestamptz NOT NULL,
    ends timestamptz DEFAULT NULL,
    text text NOT NULL,
    product varchar(38) NOT NULL,
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
    geom geometry(Polygon, 4326),
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
	PRIMARY KEY (wfo, phenomena, significance, event_number, year, id),
    CONSTRAINT fk_vtec_event
    FOREIGN KEY (wfo, phenomena, significance, event_number, year)
    REFERENCES vtec.events(wfo, phenomena, significance, event_number, year) ON DELETE CASCADE
) PARTITION BY LIST (year);
ALTER TABLE vtec.updates OWNER TO mds;
GRANT ALL ON TABLE vtec.updates TO awips_service;
GRANT SELECT ON TABLE vtec.updates TO nobody, api_service;

CREATE OR REPLACE FUNCTION vtec.CREATE_YEARLY_PARTITIONS (starts INTEGER, ends INTEGER) RETURNS VOID AS $$
BEGIN
    FOR year IN starts..ends
    LOOP
	    -- VTEC Tables
	    PERFORM create_yearly_list_partition('vtec.events', year);
        EXECUTE format('
            	CREATE INDEX vtec_event_%s_issued ON vtec.events_%s(issued);',
            	year, year);
        EXECUTE format('
            	CREATE INDEX vtec_event_%s_starts ON vtec.events_%s(starts);',
            	year, year);
        EXECUTE format('
                CREATE INDEX vtec_event_%s_expires ON vtec.events_%s(expires);',
                year, year);
        EXECUTE format('
            	CREATE INDEX vtec_event_%s_ends ON vtec.events_%s(ends);',
            	year, year);
        EXECUTE format('
            	CREATE INDEX vtec_event_%s_phenomena_significance ON vtec.events_%s(phenomena, significance);',
            	year, year);
        EXECUTE format('
            	CREATE INDEX vtec_event_%s_is_emergency ON vtec.events_%s(is_emergency) WHERE is_emergency = true;',
            	year, year);
        EXECUTE format('
            	CREATE INDEX vtec_event_%s_is_pds ON vtec.events_%s(is_pds) WHERE is_pds = true;',
            	year, year);
        EXECUTE format('
            	ALTER TABLE vtec.events_%s OWNER TO mds;',
            	year);
        EXECUTE format('
            	GRANT ALL ON TABLE vtec.events_%s TO awips_service;',
            	year);
        EXECUTE format('
            	GRANT SELECT ON TABLE vtec.events_%s TO nobody, api_service;',
            	year);

	    PERFORM create_yearly_list_partition('vtec.ugcs', year);
        EXECUTE format('
            	CREATE INDEX vtec_ugc_%s_ugc ON vtec.ugcs_%s(ugc);',
            	year, year);
        EXECUTE format('
            	CREATE INDEX vtec_ugc_%s_action ON vtec.ugcs_%s(action);',
            	year, year);
        EXECUTE format('
            	ALTER TABLE vtec.ugcs_%s OWNER TO mds;',
            	year);
        EXECUTE format('
            	GRANT ALL ON TABLE vtec.ugcs_%s TO awips_service;',
            	year);
        EXECUTE format('
            	GRANT SELECT ON TABLE vtec.ugcs_%s TO nobody, api_service;',
            	year);

	    PERFORM create_yearly_list_partition('vtec.updates', year);
        EXECUTE format('
            	CREATE INDEX vtec_update_%s_geom ON vtec.updates_%s USING GIST (geom);',
            	year, year);
        EXECUTE format('ALTER TABLE vtec.updates_%s OWNER TO mds;',
            	year);
        EXECUTE format('GRANT ALL ON TABLE vtec.updates_%s TO awips_service;',
            	year);
        EXECUTE format('GRANT SELECT ON TABLE vtec.updates_%s TO nobody, api_service;',
            	year);
    END LOOP;
END
$$ LANGUAGE PLPGSQL;

DO $$
BEGIN
    PERFORM vtec.CREATE_YEARLY_PARTITIONS(2020, 2030);
END
$$;