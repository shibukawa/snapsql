-- Blog API Database Schema

-- Drop tables if they exist
DROP TABLE IF EXISTS comments CASCADE;
DROP TABLE IF EXISTS posts CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Users table
CREATE TABLE users (
    user_id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    full_name VARCHAR(255),
    bio TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Posts table
CREATE TABLE posts (
    post_id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    author_id INTEGER NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    published BOOLEAN NOT NULL DEFAULT false,
    view_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by INTEGER REFERENCES users(user_id),
    updated_by INTEGER REFERENCES users(user_id)
);

-- Comments table
CREATE TABLE comments (
    comment_id SERIAL PRIMARY KEY,
    post_id INTEGER NOT NULL REFERENCES posts(post_id) ON DELETE CASCADE,
    author_id INTEGER NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for better query performance
CREATE INDEX idx_posts_author_id ON posts(author_id);
CREATE INDEX idx_posts_created_at ON posts(created_at DESC);
CREATE INDEX idx_comments_post_id ON comments(post_id);
CREATE INDEX idx_comments_author_id ON comments(author_id);

-- Sample data
INSERT INTO users (username, email, full_name, bio) VALUES
    ('alice', 'alice@example.com', 'Alice Smith', 'Software engineer and blogger'),
    ('bob', 'bob@example.com', 'Bob Johnson', 'Tech enthusiast'),
    ('charlie', 'charlie@example.com', 'Charlie Brown', 'Writer and developer');

INSERT INTO posts (title, content, author_id, published, created_by) VALUES
    ('Getting Started with FastAPI', 'FastAPI is a modern, fast web framework for building APIs with Python...', 1, true, 1),
    ('Introduction to PostgreSQL', 'PostgreSQL is a powerful, open source object-relational database system...', 1, true, 1),
    ('Python Async Programming', 'Asynchronous programming in Python allows you to write concurrent code...', 2, true, 2),
    ('Draft Post', 'This is a draft post that is not yet published.', 2, false, 2);

INSERT INTO comments (post_id, author_id, content) VALUES
    (1, 2, 'Great introduction! Very helpful.'),
    (1, 3, 'Thanks for sharing this.'),
    (2, 3, 'PostgreSQL is indeed powerful!'),
    (3, 1, 'Nice explanation of async/await.');
