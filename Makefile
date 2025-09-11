# Format Go code using goimports
format:
	goimports -local github.com/aliskhannn/delayed-notifier -w .

# Run linters: vet + golangci-lint
lint:
	go vet ./... && golangci-lint run ./...

# Build and start all Docker services
docker-up:
	docker compose up --build

# Stop and remove all Docker services and volumes
docker-down:
	docker compose down -v