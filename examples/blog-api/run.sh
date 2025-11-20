#!/bin/bash
# Quick start script for Blog API example
# Run this from the SnapSQL repository root

set -e

echo "🚀 Blog API Quick Start"
echo "======================="
echo ""

# Check if we're in the repository root
if [ ! -f "go.mod" ] || [ ! -d "examples/blog-api" ]; then
    echo "❌ Error: Please run this script from the SnapSQL repository root"
    exit 1
fi

# Check if uv is installed
if ! command -v uv &> /dev/null; then
    echo "❌ Error: uv is not installed"
    echo "Install it from: https://docs.astral.sh/uv/"
    exit 1
fi

# Check if docker-compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Error: docker-compose is not installed"
    exit 1
fi

echo "✓ Prerequisites check passed"
echo ""

# Step 1: Start database
echo "📦 Step 1: Starting PostgreSQL database..."
cd examples/blog-api
docker-compose up -d
cd ../..

# Wait for database to be ready
echo "⏳ Waiting for database to be ready..."
sleep 3

# Check if database is ready
for i in {1..30}; do
    if docker-compose -f examples/blog-api/docker-compose.yml exec -T postgres pg_isready -U bloguser &> /dev/null; then
        echo "✓ Database is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ Database failed to start"
        exit 1
    fi
    sleep 1
done

echo ""

# Step 2: Extract database schema
echo "📊 Step 2: Extracting database schema..."
(cd examples/blog-api && tbls doc --force --rm-dist)
echo "✓ Schema extracted"
echo ""

# Step 3: Generate Python code
echo "🔧 Step 3: Generating Python query code..."
go run ./cmd/snapsql generate \
  --config examples/blog-api/snapsql.yaml \
  --lang python \
  --output examples/blog-api/dataaccess

echo "✓ Code generated"
echo ""

# Step 4: Install dependencies
echo "📚 Step 4: Installing Python dependencies..."
cd examples/blog-api
uv pip install -r requirements.txt
echo "✓ Dependencies installed"
echo ""

# Step 5: Start server
echo "🌐 Step 5: Starting FastAPI server..."
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Blog API is starting..."
echo "  "
echo "  📖 API Documentation: http://localhost:8000/docs"
echo "  🔄 Alternative Docs:   http://localhost:8000/redoc"
echo "  ❤️  Health Check:      http://localhost:8000/health"
echo "  "
echo "  Press Ctrl+C to stop the server"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

uv run uvicorn app.main:app --reload --host 0.0.0.0 --port 8000
