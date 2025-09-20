-- PostgreSQL specific initialization
CREATE TABLE IF NOT EXISTS departments (
    id INTEGER PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description VARCHAR(512)
);

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    age INTEGER,
    status VARCHAR(50),
    department_id INTEGER
);

INSERT INTO departments (id, name, description) VALUES
    (1, 'Engineering', 'Software development'),
    (2, 'Design', 'UI/UX design'),
    (3, 'Marketing', 'Product marketing')
ON CONFLICT (id) DO NOTHING;

-- Table referenced by templated query users_/*= table_suffix */dev when table_suffix = 'test'
-- (ok_valid_query.md). We normalize to users_testdev.
CREATE TABLE IF NOT EXISTS users_testdev (
    LIKE users INCLUDING ALL
);
