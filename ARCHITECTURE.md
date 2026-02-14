# Architecture

Navidrums follows a layered architecture.

## Layers

UI → Handlers → Services → Repository
                      ↓
                   Providers
                      ↓
                   Filesystem
                      ↓
                    Tagging
                      ↓
                    Worker

---

## Layer Responsibilities

### Handlers
HTTP parsing and response formatting only.

### Services
Business workflows and orchestration.

### Repository
Persistent state and queries.

### Providers
External API adapters.

### Filesystem
All local disk IO.

### Worker
Background execution engine.

---

## Dependency Graph

handlers → services
services → repository
services → providers
services → filesystem
worker → services

Forbidden:

repository → services
providers → repository
handlers → providers
handlers → filesystem
