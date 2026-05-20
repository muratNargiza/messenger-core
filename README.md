# Messenger Core

A Go-based messenger backend service utilizing Hertz, Postgres, ScyllaDB, and Redis.

---

## 📂 Project Structure

Here is a breakdown of the core directories and their responsibilities:

* **`cmd/`**
  * `server/` — Entry point for the main HTTP & WebSocket application server.
  * `test_ws/` — A WebSocket client tool for testing real-time messaging.
* **`internal/`**
  * `config/` — Configuration loader and environment variable bindings.
  * `controller/` — REST API controllers, router definitions, and middleware (e.g., JWT Authentication).
  * `domain/` — Core business logic interfaces and system abstractions.
  * `entity/` — Domain data models.
  * `messenger/` — WebSocket client, hub, and connection upgrade handlers.
  * `pkg/` — Reusable packages and system-wide utilities (e.g., Zap logging).
  * `repository/` — Database access layer (PostgreSQL for user data, ScyllaDB/Redis for messaging and caching).
  * `usecase/` — Application orchestration layer coordinating business rules.
* **`migrations/`**
  * SQL and CQL schema migrations for PostgreSQL and ScyllaDB.

---

## 🚀 Getting Started

### Prerequisites

Make sure you have the following installed on your machine:
* **Go** (v1.26+)
* **Docker** & **Docker Compose**

### Running the Project

1. **Spin up dependencies:**
   ```bash
   docker-compose up -d
   ```
   *This starts PostgreSQL (`db`), Redis, and ScyllaDB.*

2. **Run the application server:**
   ```bash
   go run cmd/server/main.go
   ```
