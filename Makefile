.PHONY: protos build up down logs clean

# Generate protobuf files
protos:
	mkdir -p ./pkg/protos/gen
	cd ./pkg/protos && protoc --go_out=gen --go_opt=paths=source_relative --go-grpc_out=gen --go-grpc_opt=paths=source_relative match.proto

# Build Docker images
build:
	docker-compose build

# Start all services
up:
	docker-compose up -d

# Start services with build
up-build:
	docker-compose up -d --build

# Stop all services
down:
	docker-compose down

# Stop and remove volumes
down-clean:
	docker-compose down -v

# View logs
logs:
	docker-compose logs -f

# View specific service logs
logs-api:
	docker-compose logs -f api

logs-worker:
	docker-compose logs -f worker

logs-redis:
	docker-compose logs -f redis

logs-nats:
	docker-compose logs -f nats

# Restart services
restart:
	docker-compose restart

# Show service status
status:
	docker-compose ps

# Clean up everything
clean: down-clean
	docker system prune -f