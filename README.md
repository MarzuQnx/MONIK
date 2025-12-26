# MONIK-ENTERPRISE

**Enterprise monitoring system for MikroTik routers** with unified versioning and CHANGELOG management.

## 🚀 Quick Start

### Prerequisites

- Go 1.23.0+
- Git
- MikroTik RouterOS device

### Installation

1. Clone the repository:
```bash
git clone https://github.com/MarzuQnx/MONIK.git
cd MONIK-ENTERPRISE
```

2. Install dependencies:
```bash
cd backend
go mod tidy
```

3. Configure environment:
```bash
cp .env.example .env
# Edit .env with your MikroTik connection details
```

4. Run the application:
```bash
go run cmd/monik/main.go
```

## 📋 Versioning System

This project uses **Semantic Versioning (SemVer)** with **Conventional Commits** for automated version management.

### Version Format
```
MAJOR.MINOR.PATCH
```

- **MAJOR**: Breaking changes (API, database schema, architecture)
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes and optimizations

### Commit Message Format

All commits must follow the conventional commits format:

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

#### Commit Types

| Type | Description | Example |
|------|-------------|---------|
| `feat` | New feature | `feat(api): add user authentication` |
| `fix` | Bug fix | `fix(database): fix connection timeout` |
| `perf` | Performance improvement | `perf(api): optimize query performance` |
| `revert` | Revert changes | `revert(feat): revert user authentication` |
| `docs` | Documentation changes | `docs(readme): update installation guide` |
| `style` | Code formatting | `style(code): fix indentation` |
| `refactor` | Code refactoring | `refactor(api): simplify endpoint logic` |
| `test` | Test changes | `test(api): add unit tests` |
| `chore` | Maintenance tasks | `chore(deps): update dependencies` |

#### Breaking Changes

For breaking changes, add `BREAKING CHANGE:` in the commit footer:

```
feat(api): add new authentication method

BREAKING CHANGE: Old authentication method removed
```

## 📖 CHANGELOG

The CHANGELOG is automatically generated and maintained in [`CHANGELOG.md`](CHANGELOG.md). It follows the [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) format.

### CHANGELOG Categories

- **Added**: New features and functionality
- **Changed**: Changes in existing functionality
- **Fixed**: Bug fixes
- **Security**: Security-related changes
- **Tested**: Testing-related changes
- **Deprecated**: Features that will be removed

## 🏗️ Architecture

### Backend (Go)
- **Framework**: Gin for REST API
- **Database**: SQLite with GORM ORM
- **RouterOS Integration**: go-routeros library
- **Real-time**: WebSocket for live updates
- **Monitoring**: Worker pool with load balancing

### Frontend
- **Real-time dashboard** with WebSocket connection
- **Traffic visualization** charts
- **System health monitoring**
- **Interface status tracking**

### Database Schema
- **Automatic migrations** with versioning
- **Interface monitoring** tables
- **Traffic snapshot** storage
- **Counter reset** logging
- **Monthly quota** tracking

## 🔧 Configuration

### Environment Variables

```bash
# Server Configuration
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# Database Configuration
DB_PATH=data/monik.db
DB_MAX_OPEN_CONN=25
DB_MAX_IDLE_CONN=5

# Router Configuration
ROUTER_IP=172.23.192.1
ROUTER_PORT=8778
ROUTER_USERNAME=dbanie
ROUTER_PASSWORD="==gmg25"
ROUTER_TIMEOUT=30s

# Versioning Configuration
VERSIONING_ENABLED=true
VERSIONING_STRATEGY=semantic
CHANGELOG_PATH=CHANGELOG.md
APP_VERSION=1.0.0
```

## 🚀 Deployment

### CI/CD Pipeline

The project includes a GitHub Actions workflow (`.github/workflows/release.yml`) that:

1. **Detects version changes** from commit messages
2. **Updates CHANGELOG** automatically
3. **Creates Git tags** for releases
4. **Builds and tests** the application
5. **Creates GitHub releases**

### Release Process

#### Automatic (Recommended)
1. Make commits with conventional commit format
2. Push to `main` or `master` branch
3. CI/CD pipeline handles the rest

#### Manual
```bash
# Run version bump
./scripts/version-bump.sh

# Update CHANGELOG
./scripts/migrate-changelog.sh

# Create tag
git tag v1.1.0
git push origin v1.1.0
```

## 🧪 Testing

### Unit Tests
```bash
cd backend
go test ./...
```

### Versioning Tests
```bash
./scripts/test-versioning-simple.sh
```

### Integration Tests
```bash
# Test CI/CD pipeline
./scripts/test-ci-pipeline.sh

# Test multi-component release
./scripts/test-multi-component.sh
```

## 📊 Monitoring

### System Health
- **Error rate tracking** across all components
- **Response time monitoring** for API endpoints
- **Worker pool metrics** (active workers, queue size)
- **WebSocket performance** (message throughput, drop rates)

### Performance Metrics
- **Interface monitoring** for 100+ concurrent interfaces
- **<100ms response time** for 95% of requests
- **40% memory usage reduction** vs previous implementation
- **Even CPU distribution** across cores

## 🔒 Security

### Authentication
- **Input validation** for all API endpoints
- **Structured logging** for security events
- **Audit trail** for user actions
- **Secure MikroTik connections**

### Best Practices
- **No credentials in commit messages**
- **Environment-based configuration**
- **Regular dependency updates**
- **Security scanning in CI/CD**

## 📚 Documentation

- [Versioning Guide](VERSIONING.md) - Complete versioning system documentation
- [API Documentation](docs/API_DOCS.md) - REST API endpoints and usage
- [Configuration Guide](docs/CONFIGURATION.md) - Detailed configuration options
- [Troubleshooting](docs/TROUBLESHOOTING.md) - Common issues and solutions
- [Architecture Overview](docs/ARCHITECTURE.md) - System architecture details

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make commits with conventional commit format
4. Push to your fork
5. Create a pull request

### Commit Message Guidelines
- Use conventional commit format
- Be descriptive but concise
- Include scope when relevant
- Add `BREAKING CHANGE:` for breaking changes

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **MikroTik** for RouterOS and API
- **Go community** for excellent libraries
- **Semantic Release** for automated versioning
- **Keep a Changelog** for CHANGELOG standards

---

**Note**: This project uses a unified versioning system that consolidates all documentation into a single CHANGELOG, eliminating the need for multiple `.md` files while maintaining comprehensive tracking of all changes.