-- SQLite DDL for testdata/appsample
-- Tables required by queries in testdata/appsample/queries

CREATE TABLE accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT,
    status TEXT
);
