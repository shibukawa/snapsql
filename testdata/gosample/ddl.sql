-- SQLite DDL for testdata/gosample
-- Tables required by queries in testdata/gosample/queries

CREATE TABLE accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT,
    status TEXT
);
