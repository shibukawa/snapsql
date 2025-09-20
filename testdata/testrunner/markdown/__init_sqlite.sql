-- SQLite specific initialization (self-contained)
CREATE TABLE IF NOT EXISTS departments (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT
);

CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    age INTEGER,
    status TEXT,
    note TEXT,
    comment TEXT,
    department_id INTEGER
);

INSERT INTO departments (id, name, description) VALUES
    (1, 'Engineering', 'Software development'),
    (2, 'Design', 'UI/UX design'),
    (3, 'Marketing', 'Product marketing')
ON CONFLICT(id) DO NOTHING;

-- Keep users_testdev for parity even if template test removed (future reinstatement)
CREATE TABLE IF NOT EXISTS users_testdev (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    age INTEGER,
    status TEXT,
    department_id INTEGER
);
