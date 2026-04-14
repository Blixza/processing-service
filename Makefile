.PHONY: up down dev

DEPLOY_DIR = ./deployments

up:
	cd $(DEPLOY_DIR) && docker-compose up -d

down:
	cd $(DEPLOY_DIR) && docker-compose down

# api
api:
	go run cmd/api/main.go

# worker
dev:
	go run cmd/worker/main.go

# file server
fs: 
	python -m http.server 9000

# clean storages
clean:
	rm -f storage/*
	rm -f internal/transport/rabbitmq/storage/*
	rm -f internal/processor/storage/*