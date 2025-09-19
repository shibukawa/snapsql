-- MySQL specific initialization
CREATE TABLE IF NOT EXISTS departments (
    id INT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description VARCHAR(512)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    age INT,
    status VARCHAR(50),
    department_id INT
) ENGINE=InnoDB;

INSERT IGNORE INTO departments (id, name, description) VALUES
    (1, 'Engineering', 'Software development'),
    (2, 'Design', 'UI/UX design'),
    (3, 'Marketing', 'Product marketing');

-- Helper table for templated users_testdev used in ok_valid_query.md
CREATE TABLE IF NOT EXISTS users_testdev (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    age INT,
    status VARCHAR(50),
    department_id INT
) ENGINE=InnoDB;
