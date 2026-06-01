include .env
export

MIGRATION_DIR := migration
DB_URL        := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

.PHONY: migrate-up migrate-down migrate-force migrate-version install-tools

## Run all pending migrations
migrate-up:
	docker run --rm \
      --network keyloop-test-network \
      -v $$(PWD)/migration:/migration \
      migrate/migrate \
      -path=/migration \
      -verbose \
      -database "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable" up

## Roll back the last applied migration
migrate-down:
	migrate -path $(MIGRATION_DIR) -database "$(DB_URL)" down 1

## Roll back ALL migrations (destructive)
migrate-down-all:
	migrate -path $(MIGRATION_DIR) -database "$(DB_URL)" down -all

## Print current migration version
migrate-version:
	migrate -path $(MIGRATION_DIR) -database "$(DB_URL)" version

## Force-set migration version without running SQL (use after manual fix)
migrate-force:
	migrate -path $(MIGRATION_DIR) -database "$(DB_URL)" force $(V)

## Regenerate Swagger docs (fixes swag v2 → v1 import incompatibility automatically)
swagger:
	swag init -g cmd/api/main.go -o docs
	sed -i 's|"github.com/swaggo/swag/v2"|"github.com/swaggo/swag"|' docs/docs.go
	sed -i '/LeftDelim:/d; /RightDelim:/d' docs/docs.go

## Install golang-migrate CLI
install-tools:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

up: 
	docker compose -f docker-compose.yml --env-file .env up -d
	docker run --rm \
      --network keyloop-test-network \
      -v $(PWD)/migration:/migration \
      migrate/migrate \
      -path=/migration \
      -verbose \
      -database "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable" up
build:
	docker build -f Dockerfile -t keyloop-test .

down:
	docker compose -f docker-compose.yml --env-file .env down

test-domain:
	go test -tags=domain ./test/domain/... -v

test-integration:
	go test -tags=integration ./test/integration/... -v

test-usecase:
	go test -tags=usecase ./test/usecase/... -v