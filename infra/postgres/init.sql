-- Creates one database per service. Executed once, on the first start of the
-- PostgreSQL container (docker-entrypoint-initdb.d).
CREATE DATABASE auth_service;
CREATE DATABASE catalog_service;