# ============================================================
# Project
# ============================================================

PROJECT_NAME := booking-microservices

COMPOSE := docker compose

SERVICES := profile-service booking-service notification-service

GO_SERVICES := profile-service booking-service notification-service


# ============================================================
# Default target
# ============================================================

.DEFAULT_GOAL := help


# ============================================================
# Help
# ============================================================

.PHONY: help

help:
	@echo ""
	@echo " $(PROJECT_NAME)"
	@echo ""
	@echo " Infrastructure:"
	@echo "   make up             Start all services"
	@echo "   make up-build       Build and start all services"
	@echo "   make down           Stop and remove containers"
	@echo "   make stop           Stop containers"
	@echo "   make restart        Restart all services"
	@echo "   make ps             Show container status"
	@echo "   make logs           Follow all logs"
	@echo "   make logs-service   Follow logs for a service"
	@echo ""
	@echo " Build:"
	@echo "   make build          Build Docker images"
	@echo "   make rebuild        Rebuild images without cache"
	@echo ""
	@echo " Go:"
	@echo "   make test           Run tests"
	@echo "   make test-race      Run tests with race detector"
	@echo "   make vet            Run go vet"
	@echo "   make fmt            Format Go code"
	@echo "   make tidy           Run go mod tidy"
	@echo ""
	@echo " Database:"
	@echo "   make migrate        Run migrations"
	@echo "   make db-shell       Open PostgreSQL shell"
	@echo ""
	@echo " Checks:"
	@echo "   make config         Validate Compose configuration"
	@echo "   make health         Check service health endpoints"
	@echo "   make check          Run configuration and health checks"
	@echo ""
	@echo " Cleanup:"
	@echo "   make clean          Remove containers and networks"
	@echo "   make clean-volumes  Remove containers and volumes"
	@echo ""


# ============================================================
# Infrastructure
# ============================================================

.PHONY: up up-build down stop restart ps logs logs-service

up:
	$(COMPOSE) up -d

up-build:
	$(COMPOSE) up --build -d

down:
	$(COMPOSE) down

stop:
	$(COMPOSE) stop

restart:
	$(COMPOSE) restart

ps:
	$(COMPOSE) ps

logs:
	$(COMPOSE) logs -f

logs-service:
	@if [ -z "$(SERVICE)" ]; then \
		echo "Usage: make logs-service SERVICE=booking-service"; \
		exit 1; \
	fi
	$(COMPOSE) logs -f $(SERVICE)


# ============================================================
# Build
# ============================================================

.PHONY: build rebuild

build:
	$(COMPOSE) build

rebuild:
	$(COMPOSE) build --no-cache


# ============================================================
# Go tests
# ============================================================

.PHONY: test test-race

test:
	@set -e; \
	for service in $(GO_SERVICES); do \
		echo ""; \
		echo "Running tests: $$service"; \
		cd $$service && go test ./...; \
		cd ..; \
	done

test-race:
	@set -e; \
	for service in $(GO_SERVICES); do \
		echo ""; \
		echo "Running race detector: $$service"; \
		cd $$service && go test -race ./...; \
		cd ..; \
	done


# ============================================================
# Static analysis
# ============================================================

.PHONY: vet

vet:
	@set -e; \
	for service in $(GO_SERVICES); do \
		echo ""; \
		echo "Running go vet: $$service"; \
		cd $$service && go vet ./...; \
		cd ..; \
	done


# ============================================================
# Formatting
# ============================================================

.PHONY: fmt

fmt:
	@set -e; \
	for service in $(GO_SERVICES); do \
		echo "Formatting $$service"; \
		find $$service -name '*.go' -type f -exec gofmt -w {} +; \
	done


# ============================================================
# Go modules
# ============================================================

.PHONY: tidy

tidy:
	@set -e; \
	for service in $(GO_SERVICES); do \
		echo ""; \
		echo "Running go mod tidy: $$service"; \
		cd $$service && go mod tidy; \
		cd ..; \
	done


# ============================================================
# Database
# ============================================================

.PHONY: migrate db-shell

migrate:
	$(COMPOSE) up -d postgres
	$(COMPOSE) run --rm user-migrations
	$(COMPOSE) run --rm booking-migrations
	$(COMPOSE) run --rm notification-migrations

db-shell:
	$(COMPOSE) exec postgres psql -U postgres -d postgres


# ============================================================
# Configuration checks
# ============================================================

.PHONY: config

config:
	$(COMPOSE) config --quiet


# ============================================================
# Health checks
# ============================================================

.PHONY: health check

health:
	@echo "Checking user-service..."
	@curl -fsS http://localhost:8081/health/ready
	@echo ""
	@echo "Checking booking-service..."
	@curl -fsS http://localhost:8083/health/ready
	@echo ""
	@echo "Checking notification-service..."
	@curl -fsS http://localhost:8082/health/ready
	@echo ""
	@echo "All services are healthy."

check: config health


# ============================================================
# Cleanup
# ============================================================

.PHONY: clean clean-volumes

clean:
	$(COMPOSE) down --remove-orphans

clean-volumes:
	$(COMPOSE) down --remove-orphans --volumes