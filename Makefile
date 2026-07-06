.Phony: build run test docker-run clean 

build:
	go build -o bun/agent ./cmd/agent

run:
	go run ./cmd/agent

test:
	go test ./... -v

docker-run:
	docker-compose -f deployments/docker-compose.yml up --build

clean:
	rm -rf bin/

docker-build:
	docker-compose -f deployments/docker-compose.yml build
	
docker-run:
	docker-compose -f deployments/docker-compose.yml --env-file .env up -d

docker-stop:
	docker-compose -f deployments/docker-compose.yml --env-file .env down

docker-logs:
	docker-compose -f deployments/docker-compose.yml logs -f

docker-restart:
	docker-compose -f deployments/docker-compose.yml restart