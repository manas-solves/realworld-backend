# RealWorld Backend - Go Implementation

[![CI](https://github.com/manas-solves/realworld-backend/actions/workflows/CI.yml/badge.svg?branch=main)](https://github.com/manas-solves/realworld-backend/actions/workflows/CI.yml)
[![Secret Scanning](https://github.com/manas-solves/realworld-backend/actions/workflows/secret-scanning.yml/badge.svg?branch=main)](https://github.com/manas-solves/realworld-backend/actions/workflows/secret-scanning.yml)
[![codecov](https://codecov.io/gh/96malhar/realworld-backend/branch/main/graph/badge.svg?token=JS1YCVM98M)](https://codecov.io/gh/96malhar/realworld-backend)
[![Go Report Card](https://goreportcard.com/badge/github.com/manas-solves/realworld-backend)](https://goreportcard.com/report/github.com/manas-solves/realworld-backend)

> A production-ready social blogging platform backend (like Medium.com) built with Go, featuring clean architecture, comprehensive testing, and optimized queries.

This project adheres to the [RealWorld](https://realworld-docs.netlify.app/) API specification, demonstrating modern Go development practices including JWT authentication, proper error handling, and 80%+ test coverage.

## Table of Contents

- [Features](#features)
- [Tech Stack](#tech-stack)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Development](#development)
  - [Project Structure](#project-structure)
  - [Running the Server](#running-the-server)
  - [Task Runner](#task-runner)
- [API Documentation](#api-documentation)
- [Testing](#testing)
- [Configuration](#configuration)
- [CI/CD](#cicd)
- [Contributing](#contributing)

## Features

### ✅ Implemented Endpoints

- **Authentication & Authorization**
  - User registration and login
  - JWT-based authentication
  - Get/Update current user profile

- **Articles**
  - Create, read, update, and delete articles
  - List articles with filtering (by tag, author, favorited by user)
  - Pagination support (limit/offset)
  - Feed of articles from followed users
  - Favorite/unfavorite articles
  - Article slugs with automatic generation

- **Comments**
  - Add comments to articles
  - Get all comments for an article
  - Delete comments

- **User Profiles**
  - View user profiles
  - Follow/unfollow users

- **Tags**
  - List all tags (with persistence)

### 🚀 Key Features

- **Clean Architecture**: Separation of concerns with handlers, data layer, and business logic
- **Optimized Queries**: Efficient single-query operations for common use cases
- **Comprehensive Testing**: 150 unit tests + Newman API integration tests
- **Input Validation**: Request validation with custom validator package
- **Error Handling**: Structured error responses with proper HTTP status codes
- **Graceful Shutdown**: Clean server shutdown with context cancellation

## Tech Stack

- **Language**: Go 1.25+
- **Web Framework**: [chi](https://github.com/go-chi/chi) - Lightweight, idiomatic HTTP router
- **Database**: PostgreSQL 15+ with [pgx](https://github.com/jackc/pgx) driver
- **Authentication**: JWT tokens with [golang-jwt](https://github.com/golang-jwt/jwt)
- **Query Builder**: [Squirrel](https://github.com/Masterminds/squirrel) for dynamic SQL
- **Testing**: Standard `testing` package + [testify](https://github.com/stretchr/testify)
- **Task Runner**: [Task](https://taskfile.dev/) for development workflows
- **API Testing**: [Newman](https://www.npmjs.com/package/newman) (Postman CLI)

## Prerequisites

- **Go** 1.25 or higher
- **PostgreSQL** 15 or higher
- **Docker** & Docker Compose (optional, for Postgres)
- **Task** (optional but recommended): `brew install go-task`
- **Node.js** (for Newman API tests): `brew install node`

## Quick Start

```bash
# Clone the repository
git clone https://github.com/manas-solves/realworld-backend.git
cd realworld-backend

# Install dependencies
task install:dependencies

# Start the server (automatically starts Postgres, runs migrations)
task server:start

# In another terminal, verify it's running
curl http://localhost:4000/healthcheck
```

The API will be available at `http://localhost:4000`.

## Development

### Project Structure

<details>
<summary>Click to expand</summary>

```
.
├── cmd/api/                    # Application entry point
│   ├── main.go                # Server initialization
│   ├── routes.go              # Route definitions
│   ├── handlers (*.go)        # HTTP handlers
│   ├── middleware.go          # Authentication, recovery, etc.
│   ├── *_test.go              # Integration tests
│   └── openapi.yml            # API specification
├── internal/
│   ├── auth/                  # JWT token generation & validation
│   ├── data/                  # Data models and operations
│   │   ├── articles.go        # Article CRUD, favorites, feed
│   │   ├── users.go           # User management, authentication
│   │   ├── comments.go        # Comment operations
│   │   ├── tags.go            # Tag management
│   │   └── store.go           # Store interfaces and initialization
│   ├── validator/             # Input validation utilities
│   └── vcs/                   # Version information
├── migrations/                 # Database schema migrations
├── docker-compose.yml         # PostgreSQL container definition
└── Taskfile.yml               # Development task definitions
```

</details>

### Database Schema

<details>
<summary>Click to expand</summary>

The database schema consists of 6 main tables with the following relationships:

```mermaid
erDiagram
    users ||--o{ articles : "authors"
    users ||--o{ comments : "writes"
    users ||--o{ favorites : "favorites"
    users ||--o{ follows : "follower"
    users ||--o{ follows : "followed"
    articles ||--o{ comments : "has"
    articles ||--o{ favorites : "favorited_by"
    
    users {
        bigserial id PK
        citext username UK
        citext email UK
        bytea password_hash
        text bio
        text image
        integer version
    }
    
    articles {
        serial id PK
        varchar slug UK
        varchar title
        text description
        text body
        text_array tag_list
        timestamp created_at
        timestamp updated_at
        integer favorites_count
        integer author_id FK
        integer version
    }
    
    comments {
        serial id PK
        text body
        integer article_id FK
        bigint author_id FK
        timestamp created_at
        timestamp updated_at
    }
    
    favorites {
        integer user_id PK_FK
        integer article_id PK_FK
    }
    
    follows {
        integer follower_id PK_FK
        integer followed_id PK_FK
    }
    
    tags {
        serial id PK
        varchar tag UK
    }
```

**Key Relationships:**
- **users → articles**: One-to-many (a user can create many articles)
- **users → comments**: One-to-many (a user can write many comments)
- **users ↔ users** (via follows): Many-to-many (users can follow each other)
- **users ↔ articles** (via favorites): Many-to-many (users can favorite many articles)
- **articles → comments**: One-to-many (an article can have many comments)
- **tags**: Standalone table for tag persistence (articles store tags in `tag_list` array)

**Indexes:**
- Articles: `slug`, `author_id`, `created_at`, `tag_list` (GIN index)
- Comments: `article_id`, `author_id`, `created_at`
- Favorites: `user_id`, `article_id`
- Tags: `tag`
- Follows: Composite primary key on `(follower_id, followed_id)`

</details>

### Running the Server

<details>
<summary>Click to expand</summary>

#### Quick Start (Recommended)

```bash
# Starts Postgres + runs migrations + starts server
task server:start
```

#### Manual Start

```bash
# Build the binary
task server:build

# Run with custom configuration
./bin/api \
  -port 8080 \
  -env production \
  -db-dsn $DB_DSN \
  -jwt-secret $JWT_SECRET
```

#### Configuration Options

```bash
./bin/api -help

Flags:
  -port int
        API server port (default 4000)
  -env string
        Environment (development|staging|production) (default "development")
  -db-dsn string
        PostgreSQL DSN (required)
  -db-max-open-conns int
        PostgreSQL max open connections (default 25)
  -db-max-idle-time duration
        PostgreSQL max connection idle time (default 15m)
  -jwt-secret string
        JWT secret key (required)
  -jwt-issuer string
        JWT issuer (default "realworld-api")
```

</details>

### Task Runner

<details>
<summary>Click to expand</summary>

This project uses [Task](https://taskfile.dev/) for managing common development workflows. All tasks are defined in `Taskfile.yml`.

#### Key Tasks

**Development**
```bash
# Start the server (auto-starts Postgres, runs migrations)
task server:start

# Build the binary
task server:build

# View all available command-line flags
task server:help
```

**Database Management**
```bash
# Start Postgres container
task docker:postgres:start

# Stop Postgres container
task docker:postgres:stop

# Create a new migration
task db:migrations:new -- create_some_table

# Run migrations
task db:migrations:up

# Rollback migrations
task db:migrations:down

# Setup database (create)
task db:setup

# Teardown database (drop)
task db:teardown
```

**Testing & Quality**
```bash
# Run all unit tests with coverage
task test

# Visualize coverage report
task visualize:coverage

# Run Newman API integration tests
task newman

# Run linters, formatters, and security checks
task audit
```

**Dependencies**
```bash
# Install required tools (migrate, golangci-lint, etc.)
task install:dependencies

# Update all Go dependencies
task update:dependencies
```

#### Environment Variables

All tasks automatically use environment variables defined in `Taskfile.yml`:

```yaml
env:
  DB_DSN: postgres://postgres:postgres@localhost:5432/conduit?sslmode=disable
  JWT_SECRET: some-secret
  JWT_ISSUER: conduit
```

Edit these values in `Taskfile.yml` to customize your local development environment.

</details>

## API Documentation

Full API specification: [OpenAPI YAML](./cmd/api/openapi.yml)

<details>
<summary><strong>Authentication Endpoints</strong></summary>

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/users` | Register new user | No |
| POST | `/users/login` | Login user | No |
| GET | `/user` | Get current user | Yes |
| PUT | `/user` | Update user | Yes |

</details>

<details>
<summary><strong>Article Endpoints</strong></summary>

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| GET | `/articles` | List articles (filterable, paginated) | No |
| GET | `/articles/feed` | Get feed from followed users | Yes |
| POST | `/articles` | Create article | Yes |
| GET | `/articles/:slug` | Get article by slug | No |
| PUT | `/articles/:slug` | Update article | Yes (author only) |
| DELETE | `/articles/:slug` | Delete article | Yes (author only) |
| POST | `/articles/:slug/favorite` | Favorite article | Yes |
| DELETE | `/articles/:slug/favorite` | Unfavorite article | Yes |

**Query Parameters for List Articles:**
- `tag` - Filter by tag name
- `author` - Filter by author username
- `favorited` - Filter by username who favorited
- `limit` - Max articles to return (default: 20, max: 100)
- `offset` - Number of articles to skip (default: 0)

</details>

<details>
<summary><strong>Comment Endpoints</strong></summary>

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/articles/:slug/comments` | Add comment to article | Yes |
| GET | `/articles/:slug/comments` | Get comments for article | No |
| DELETE | `/articles/:slug/comments/:id` | Delete comment | Yes (author only) |

</details>

<details>
<summary><strong>Profile & Tag Endpoints</strong></summary>

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| GET | `/profiles/:username` | Get user profile | No |
| POST | `/profiles/:username/follow` | Follow user | Yes |
| DELETE | `/profiles/:username/follow` | Unfollow user | Yes |
| GET | `/tags` | Get all tags | No |

</details>

## Testing

```bash
# Run all tests
task test

# Visualize code coverage (run after 'task test')
task visualize:coverage

# Run Newman API integration tests (full RealWorld spec)
task newman

# Run linting and security checks
task audit
```

### Test Coverage

- **148 unit tests** covering handlers, data layer, and authentication
- **311 Newman assertions** validating full API compliance
- **77%+ code coverage** across the application layer

### Writing Tests

Tests use the standard Go testing package with testify for assertions:

```go
func TestCreateArticle(t *testing.T) {
    t.Parallel()
    
    ts := newTestServer(t)
    token := loginUser(t, ts, "user@example.com", "password")
    
    // Test implementation...
}
```

## Configuration

### Environment Variables

For **local development**, environment variables are managed via `Taskfile.yml`:

```yaml
env:
  DB_DSN: postgres://postgres:postgres@localhost:5432/conduit?sslmode=disable
  JWT_SECRET: some-secret
  JWT_ISSUER: conduit
```

These variables are automatically applied when running tasks like `task server:start` or `task test`.

To customize local values, edit the `env:` section in `Taskfile.yml`.

### Production Configuration

For production, set environment variables directly (never commit secrets):

```bash
export DB_DSN="postgres://user:pass@host:5432/conduit?sslmode=disable"
export JWT_SECRET="strong-random-secret"
export JWT_ISSUER="your-app-name"
```

## CI/CD

GitHub Actions workflows run on every push and pull request:

### Main CI Pipeline
1. **Linting**: `golangci-lint` with strict rules
2. **Security**: `govulncheck` for vulnerability scanning
3. **Unit Tests**: Full test suite with coverage reporting
4. **Integration Tests**: Newman tests against RealWorld API spec
5. **Coverage**: Automatic upload to Codecov

See [.github/workflows/CI.yml](./.github/workflows/CI.yml) for details.

### Secret Scanning
Automated secret detection using [Gitleaks](https://github.com/gitleaks/gitleaks) to prevent accidental exposure of sensitive data like API keys, passwords, and tokens.

See [.github/workflows/secret-scanning.yml](./.github/workflows/secret-scanning.yml) for details.

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Ensure all tests pass (`task test`)
5. Run linting (`task audit`)
6. Commit your changes with meaningful messages
7. Push to your branch
8. Open a Pull Request

### Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go) guidelines
- Use `gofmt` for formatting (automatic with `task audit`)
- Write table-driven tests where appropriate
- Document exported functions and types

## Acknowledgments

- [RealWorld](https://realworld-docs.netlify.app/) - API specification and inspiration
- [Let's Go Further](https://lets-go-further.alexedwards.net/) - Architecture patterns and best practices

## License

This project is open source and available under the MIT License.

---

**Built with ❤️ using Go**
