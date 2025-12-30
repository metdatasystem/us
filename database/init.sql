-- Owner
CREATE USER mds;

-- Service users
CREATE USER awips_service;
CREATE USER api_service;

-- Script users
CREATE USER postgis;

-- Nobody
CREATE USER nobody;

CREATE DATABASE mds;

\connect mds