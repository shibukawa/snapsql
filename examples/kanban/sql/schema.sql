PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS boards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    archived_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS lists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    board_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    stage_order INTEGER NOT NULL DEFAULT 0,
    position REAL NOT NULL,
    is_archived INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (board_id) REFERENCES boards (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_lists_board_position ON lists (board_id, position);

CREATE TABLE IF NOT EXISTS list_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    stage_order INTEGER NOT NULL UNIQUE,
    is_active INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS cards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    list_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    position REAL NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (list_id) REFERENCES lists (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_cards_list_position ON cards (list_id, position);

CREATE TABLE IF NOT EXISTS card_comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    card_id INTEGER NOT NULL,
    body TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (card_id) REFERENCES cards (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_card_comments_card_created_at ON card_comments (card_id, created_at);

CREATE UNIQUE INDEX IF NOT EXISTS ux_boards_active ON boards(status) WHERE status = 'active';

INSERT INTO list_templates (name, stage_order, is_active)
VALUES
    ('Backlog', 1, 1),
    ('In Progress', 2, 1),
    ('Review', 3, 1),
    ('Done', 4, 1)
ON CONFLICT(stage_order) DO UPDATE SET
    name = excluded.name,
    is_active = excluded.is_active;
