# Blog API - FastAPI + SnapSQL Python Example

This is a sample FastAPI application demonstrating the SnapSQL Python code generator.

## Features

- FastAPI REST API for a simple blog system
- PostgreSQL database with asyncpg
- SnapSQL-generated Python query functions
- User authentication and authorization
- Blog posts with comments
- Error handling with enhanced SnapSQL errors
- Query logging and monitoring

## Project Structure

```
blog-api/
├── README.md                      # This file
├── requirements.txt               # Python dependencies
├── docker-compose.yml             # PostgreSQL database setup
├── snapsql.yaml                   # SnapSQL configuration
├── schema.sql                     # Database schema
├── queries/                       # SQL query definitions (.snap.md format)
│   ├── user_create.snap.md
│   ├── user_get.snap.md
│   ├── user_list.snap.md
│   ├── post_create.snap.md
│   ├── post_get.snap.md
│   ├── post_list.snap.md
│   ├── comment_create.snap.md
│   └── comment_list_by_post.snap.md
├── dataaccess/                    # Generated Python code (auto-generated)
│   ├── __init__.py
│   └── *.py                       # One module per query
├── app/                           # Application code
│   ├── __init__.py
│   ├── main.py                   # FastAPI application
│   ├── models.py                 # Pydantic models
│   ├── database.py               # Database connection
│   └── routers/                  # API routers
│       ├── __init__.py
│       ├── users.py
│       ├── posts.py
│       └── comments.py
└── tests/                         # Tests
    └── test_api.py
```

## Quick Start (from repository root)

This guide assumes you're starting from the SnapSQL repository root directory.

### Prerequisites

- [uv](https://docs.astral.sh/uv/) - Fast Python package installer
- Docker and Docker Compose
- Go 1.21+

### One-Command Start

The easiest way to get started:

```bash
# From repository root
./examples/blog-api/run.sh
```

This script will:
1. Start the PostgreSQL database
2. Generate Python query code
3. Install dependencies with uv
4. Start the FastAPI server

### Manual Setup

### 1. Start Database

```bash
cd examples/blog-api
docker-compose up -d
cd ../..  # Return to repository root
```

Wait for PostgreSQL to be ready:
```bash
docker-compose -f examples/blog-api/docker-compose.yml logs -f postgres
# Wait for "database system is ready to accept connections"
# Press Ctrl+C to exit logs
```

### 2. Generate Python Query Code

```bash
# From repository root
go run ./cmd/snapsql generate \
  --config examples/blog-api/snapsql.yaml \
  --lang python \
  --output examples/blog-api/dataaccess
```

### 3. Install Python Dependencies with uv

```bash
cd examples/blog-api

# Install dependencies using uv (creates virtual environment automatically)
uv pip install -r requirements.txt

# Or use uv sync if you have a pyproject.toml
# uv sync
```

### 4. Run the FastAPI Application

```bash
# Run with uv (automatically uses the virtual environment)
uv run uvicorn app.main:app --reload --host 0.0.0.0 --port 8000
```

The API will be available at:
- **API Server**: http://localhost:8000
- **Interactive API Docs**: http://localhost:8000/docs
- **Alternative Docs**: http://localhost:8000/redoc

## Alternative Setup (Traditional Method)

If you prefer not to use uv:

### 1. Install Dependencies

```bash
cd examples/blog-api

# Create virtual environment
python3 -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install dependencies
pip install -r requirements.txt
```

### 2. Start Database

```bash
docker-compose up -d
```

### 3. Generate Query Code

```bash
# From the snapsql root directory
cd ../..
go run ./cmd/snapsql generate \
  --config examples/blog-api/snapsql.yaml \
  --lang python \
  --output examples/blog-api/dataaccess
```

### 4. Run Application

```bash
cd examples/blog-api

# Development mode with auto-reload
uvicorn app.main:app --reload --host 0.0.0.0 --port 8000
```

## API Endpoints

### Users

- `POST /users` - Create a new user
- `GET /users/{user_id}` - Get user by ID
- `GET /users` - List all users
- `PUT /users/{user_id}` - Update user
- `DELETE /users/{user_id}` - Delete user

### Posts

- `POST /posts` - Create a new post
- `GET /posts/{post_id}` - Get post by ID
- `GET /posts` - List all posts (with pagination)
- `GET /posts/user/{user_id}` - Get posts by user
- `PUT /posts/{post_id}` - Update post
- `DELETE /posts/{post_id}` - Delete post

### Comments

- `POST /comments` - Create a new comment
- `GET /comments/post/{post_id}` - Get comments for a post
- `DELETE /comments/{comment_id}` - Delete comment

## Example Usage

### Create a User

```bash
curl -X POST http://localhost:8000/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john_doe",
    "email": "john@example.com",
    "full_name": "John Doe"
  }'
```

### Create a Post

```bash
curl -X POST http://localhost:8000/posts \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My First Post",
    "content": "This is the content of my first blog post.",
    "author_id": 1
  }'
```

### Get All Posts

```bash
curl http://localhost:8000/posts
```

### Add a Comment

```bash
curl -X POST http://localhost:8000/comments \
  -H "Content-Type: application/json" \
  -d '{
    "post_id": 1,
    "author_id": 1,
    "content": "Great post!"
  }'
```

## SnapSQL Features Demonstrated

1. **Response Affinity**
   - `one` - Single record queries (get user by ID)
   - `many` - Multiple records with async generators (list posts)
   - `none` - Mutations (create, update, delete)

2. **Error Handling**
   - NotFoundError for missing records
   - ValidationError for invalid parameters
   - DatabaseError for database failures
   - UnsafeQueryError for UPDATE/DELETE without WHERE

3. **System Columns**
   - `created_at`, `updated_at` timestamps
   - `created_by`, `updated_by` user tracking

4. **Query Logging**
   - Automatic query logging
   - Slow query detection
   - Error tracking

5. **Mock Mode**
   - Test without database
   - Mock data for development

## Testing the API

### Using curl

#### Automated smoke test

There is a convenience script that exercises the health check plus the user CRUD flow and hits the (currently stubbed) posts/comments routes. It defaults to `http://localhost:8000`, but you can point it elsewhere with `API_BASE_URL`.

```bash
cd examples/blog-api
API_BASE_URL=http://localhost:8000 ./scripts/curl_smoketest.sh
```

The script will:

1. Call `/health`, `/`, and `/users/`.
2. Create a unique `smoketest-<timestamp>` user and fetch it back.
3. Issue GET requests to `/posts/` and `/comments/post/1` (these still return `501 Not Implemented` until the generated handlers are wired up).

Create a user:
```bash
curl -X POST http://localhost:8000/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "email": "alice@example.com",
    "full_name": "Alice Smith",
    "bio": "Software engineer"
  }'
```

List users:
```bash
curl http://localhost:8000/users
```

Create a post:
```bash
curl -X POST "http://localhost:8000/posts?author_id=1" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My First Post",
    "content": "This is my first blog post!",
    "published": true
  }'
```

### Using the Interactive API Docs

Visit http://localhost:8000/docs to use the Swagger UI for testing all endpoints interactively.

## Development

### Regenerate Queries

After modifying `.snap.md` query files:

```bash
# From repository root
go run ./cmd/snapsql generate \
  --config examples/blog-api/snapsql.yaml \
  --lang python \
  --output examples/blog-api/dataaccess
```

### Database Migrations

For schema changes, update `schema.sql` and reapply:

```bash
# From examples/blog-api directory
docker-compose exec postgres psql -U bloguser -d blogdb -f /docker-entrypoint-initdb.d/schema.sql

# Or from host (if psql is installed)
psql -h localhost -U bloguser -d blogdb -f schema.sql
# Password: blogpass
```

### Running Tests

```bash
# From examples/blog-api directory
uv run pytest tests/ -v

# With coverage
uv run pytest --cov=app tests/
```

## Production Deployment

1. Set environment variables:
   ```bash
   export DATABASE_URL="postgresql://user:pass@host:5432/dbname"
   export SECRET_KEY="your-secret-key"
   ```

2. Run with production server:
   ```bash
   gunicorn app.main:app -w 4 -k uvicorn.workers.UvicornWorker
   ```

## License

MIT
