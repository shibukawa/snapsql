-- Notification management schema for PostgreSQL

-- Users table
CREATE TABLE users (
    id VARCHAR(10) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT NOW(),
    created_by VARCHAR(255),
    updated_at TIMESTAMP DEFAULT NOW(),
    updated_by VARCHAR(255)
);

-- Notifications table
CREATE TABLE notifications (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    icon_url TEXT,
    important BOOLEAN DEFAULT FALSE,
    cancelable BOOLEAN DEFAULT FALSE,
    cancel_message TEXT,
    canceled_at TIMESTAMP,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    created_by VARCHAR(255),
    updated_at TIMESTAMP DEFAULT NOW(),
    updated_by VARCHAR(255)
);

-- Inbox table (tracks which users have received/read which notifications)
CREATE TABLE inbox (
    notification_id INTEGER NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    user_id VARCHAR(10) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    read_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    created_by VARCHAR(255),
    updated_at TIMESTAMP DEFAULT NOW(),
    updated_by VARCHAR(255),
    PRIMARY KEY (notification_id, user_id)
);

-- Indexes for performance
CREATE INDEX idx_notifications_created_at ON notifications(created_at DESC);
CREATE INDEX idx_notifications_not_canceled ON notifications(id) WHERE canceled_at IS NULL;
CREATE INDEX idx_inbox_user ON inbox(user_id);

-- Sample data
INSERT INTO users (id, name, email) VALUES
    ('EMP001', 'Alice', 'alice@example.com'),
    ('EMP002', 'Bob', 'bob@example.com'),
    ('EMP003', 'Charlie', 'charlie@example.com');
