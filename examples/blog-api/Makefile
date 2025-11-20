.PHONY: help setup db-up db-down db-reset generate install run test clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

setup: ## Complete setup (database + generate + install)
	@echo "🚀 Setting up Blog API..."
	@$(MAKE) db-up
	@sleep 3
	@$(MAKE) generate
	@$(MAKE) install
	@echo "✅ Setup complete! Run 'make run' to start the server"

db-up: ## Start PostgreSQL database
	@echo "📦 Starting database..."
	@docker-compose up -d
	@echo "⏳ Waiting for database to be ready..."
	@sleep 3

db-down: ## Stop PostgreSQL database
	@echo "🛑 Stopping database..."
	@docker-compose down

db-reset: ## Reset database (stop, remove volumes, start)
	@echo "🔄 Resetting database..."
	@docker-compose down -v
	@docker-compose up -d
	@sleep 3

schema: ## Extract database schema using tbls
	@echo "📊 Extracting database schema..."
	@tbls doc --force --rm-dist
	@echo "✅ Schema extracted to dbdoc/"

generate: schema ## Generate Python query code from .snap.md files
	@echo "🔧 Generating Python code..."
	@cd ../.. && go run ./cmd/snapsql generate \
		--config examples/blog-api/snapsql.yaml \
		--lang python \
		--output examples/blog-api/dataaccess
	@echo "✅ Code generated"

install: ## Install Python dependencies with uv
	@echo "📚 Installing dependencies..."
	@uv pip install -r requirements.txt
	@echo "✅ Dependencies installed"

run: ## Run the FastAPI development server
	@echo "🌐 Starting FastAPI server..."
	@uv run uvicorn app.main:app --reload --host 0.0.0.0 --port 8000

test: ## Run tests
	@echo "🧪 Running tests..."
	@uv run pytest tests/ -v

test-cov: ## Run tests with coverage
	@echo "🧪 Running tests with coverage..."
	@uv run pytest --cov=app tests/ -v

clean: ## Clean generated files and cache
	@echo "🧹 Cleaning..."
	@rm -rf dataaccess
	@rm -rf __pycache__ app/__pycache__ app/routers/__pycache__
	@rm -rf .pytest_cache
	@rm -rf .coverage
	@find . -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null || true
	@echo "✅ Cleaned"

logs: ## Show database logs
	@docker-compose logs -f postgres

psql: ## Connect to database with psql
	@docker-compose exec postgres psql -U bloguser -d blogdb

all: setup run ## Complete setup and run
