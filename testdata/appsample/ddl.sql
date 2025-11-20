-- SQLite DDL for testdata/appsample
-- Tables required by queries in testdata/appsample/queries

CREATE TABLE accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT,
    status TEXT
);

CREATE TABLE posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL REFERENCES accounts(id),
    title TEXT NOT NULL,
    body TEXT,
    published_at TEXT
);
