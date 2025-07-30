# Installation Guide

This guide covers different ways to install and set up SnapSQL.

## Prerequisites

- Go 1.24 or later (for building from source)
- Database server (PostgreSQL, MySQL, or SQLite)

## Installation Methods

### 1. Install from Source (Recommended)

```bash
# Install the latest version
go install github.com/shibukawa/snapsql@latest

# Verify installation
snapsql version
```

### 2. Download Binary (Planned)

Pre-built binaries will be available for download:

```bash
# Linux/macOS
curl -L https://github.com/shibukawa/snapsql/releases/latest/download/snapsql-linux-amd64 -o snapsql
chmod +x snapsql
sudo mv snapsql /usr/local/bin/

# Windows
# Download from GitHub releases page
```

### 3. Docker (Planned)

```bash
# Run with Docker
docker run --rm -v $(pwd):/workspace shibukawa/snapsql:latest generate

# Create alias for convenience
alias snapsql='docker run --rm -v $(pwd):/workspace shibukawa/snapsql:latest'
```

### 4. Package Managers (Planned)

```bash
# Homebrew (macOS/Linux)
brew install shibukawa/tap/snapsql

# Chocolatey (Windows)
choco install snapsql

# Snap (Linux)
snap install snapsql
```

## Database Setup

### PostgreSQL

1. **Install PostgreSQL**:
   ```bash
   # Ubuntu/Debian
   sudo apt-get install postgresql postgresql-contrib
   
   # macOS
   brew install postgresql
   
   # Start service
   sudo systemctl start postgresql  # Linux
   brew services start postgresql   # macOS
   ```

2. **Create database and user**:
   ```sql
   -- Connect as postgres user
   sudo -u postgres psql
   
   -- Create database
   CREATE DATABASE myapp_dev;
   
   -- Create user
   CREATE USER myapp_user WITH PASSWORD 'myapp_password';
   
   -- Grant permissions
   GRANT ALL PRIVILEGES ON DATABASE myapp_dev TO myapp_user;
   ```

3. **Connection string**:
   ```
   postgres://myapp_user:myapp_password@localhost:5432/myapp_dev
   ```

### MySQL

1. **Install MySQL**:
   ```bash
   # Ubuntu/Debian
   sudo apt-get install mysql-server
   
   # macOS
   brew install mysql
   
   # Start service
   sudo systemctl start mysql  # Linux
   brew services start mysql   # macOS
   ```

2. **Create database and user**:
   ```sql
   -- Connect as root
   mysql -u root -p
   
   -- Create database
   CREATE DATABASE myapp_dev;
   
   -- Create user
   CREATE USER 'myapp_user'@'localhost' IDENTIFIED BY 'myapp_password';
   
   -- Grant permissions
   GRANT ALL PRIVILEGES ON myapp_dev.* TO 'myapp_user'@'localhost';
   FLUSH PRIVILEGES;
   ```

3. **Connection string**:
   ```
   myapp_user:myapp_password@tcp(localhost:3306)/myapp_dev
   ```

### SQLite

1. **Install SQLite**:
   ```bash
   # Ubuntu/Debian
   sudo apt-get install sqlite3
   
   # macOS
   brew install sqlite
   ```

2. **Create database**:
   ```bash
   # Create database file
   sqlite3 myapp_dev.db
   ```

3. **Connection string**:
   ```
   ./myapp_dev.db
   ```

## Project Setup

### 1. Initialize New Project

```bash
# Create new project
snapsql init my-project
cd my-project
```

This creates the following structure:
```
my-project/
├── snapsql.yaml          # Configuration file
├── queries/              # SQL templates
│   └── users.snap.sql    # Sample template
├── params.json           # Sample parameters
├── constants.yaml        # Project constants
└── README.md            # Project documentation
```

### 2. Configure Database

Edit `snapsql.yaml`:

```yaml
name: "my-project"
version: "1.0.0"

database:
  default_driver: "postgres"
  connection_string: "postgres://user:password@localhost:5432/mydb"
  timeout: "30s"

paths:
  queries: "./queries"
  generated: "./generated"
```

### 3. Test Configuration

```bash
# Test database connection
snapsql config test-db

# Validate configuration
snapsql config validate
```

### 4. Generate Intermediate Files

```bash
# Generate from templates
snapsql generate

# Verify generation
ls generated/
```

### 5. Test Query

```bash
# Dry run to test template
snapsql query queries/users.snap.sql --dry-run --params-file params.json

# Execute query
snapsql query queries/users.snap.sql --params-file params.json
```

## Development Environment

### VS Code Setup

1. **Install Go extension**
2. **Create `.vscode/settings.json`**:
   ```json
   {
     "go.toolsManagement.checkForUpdates": "local",
     "go.useLanguageServer": true,
     "files.associations": {
       "*.snap.sql": "sql"
     }
   }
   ```

3. **Create `.vscode/tasks.json`** for build tasks:
   ```json
   {
     "version": "2.0.0",
     "tasks": [
       {
         "label": "snapsql generate",
         "type": "shell",
         "command": "snapsql generate",
         "group": "build",
         "presentation": {
           "echo": true,
           "reveal": "always"
         }
       }
     ]
   }
   ```

### Git Setup

Create `.gitignore`:
```gitignore
# Generated files
generated/
*.db
*.log

# Environment files
.env
.env.local

# IDE files
.vscode/
.idea/
*.swp
*.swo

# OS files
.DS_Store
Thumbs.db
```

## Environment Variables

Set up environment variables for different environments:

### Development (.env.dev)
```bash
DATABASE_URL=postgres://dev:dev@localhost:5432/myapp_dev
SNAPSQL_CONFIG=snapsql.dev.yaml
SNAPSQL_VERBOSE=true
```

### Production (.env.prod)
```bash
DATABASE_URL=postgres://user:password@prod-server:5432/myapp
SNAPSQL_CONFIG=snapsql.prod.yaml
SNAPSQL_QUIET=true
```

Load environment variables:
```bash
# Load development environment
source .env.dev
snapsql query queries/users.snap.sql

# Or use with specific config
snapsql --config snapsql.prod.yaml query queries/users.snap.sql
```

## Troubleshooting

### Common Issues

1. **Command not found**:
   ```bash
   # Check if Go bin is in PATH
   echo $PATH | grep $(go env GOPATH)/bin
   
   # Add to PATH if missing
   export PATH=$PATH:$(go env GOPATH)/bin
   ```

2. **Database connection failed**:
   ```bash
   # Test connection manually
   psql "postgres://user:password@localhost:5432/dbname"
   
   # Check if service is running
   sudo systemctl status postgresql
   ```

3. **Permission denied**:
   ```bash
   # Check file permissions
   ls -la snapsql.yaml
   
   # Fix permissions
   chmod 644 snapsql.yaml
   ```

4. **Template not found**:
   ```bash
   # Check file exists
   ls -la queries/
   
   # Use absolute path
   snapsql query /full/path/to/template.snap.sql
   ```

### Getting Help

- Check command help: `snapsql --help`
- Validate configuration: `snapsql config validate`
- Enable verbose output: `snapsql --verbose <command>`
- Check GitHub issues: [https://github.com/shibukawa/snapsql/issues](https://github.com/shibukawa/snapsql/issues)

## Next Steps

After installation:

1. Read the [Template Syntax](template-syntax.md) guide
2. Learn about [Configuration](configuration.md) options
3. Explore [CLI Commands](cli-commands.md)
4. Check the [Development Guide](development.md) for contributing
