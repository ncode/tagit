.PHONY: test test-unit test-integration test-integration-ci consul-up consul-down build clean

# Default target
all: test build

# Build the binary
build:
	go build -o tagit .

# Run all tests (unit only by default)
test: test-unit

# Run unit tests
test-unit:
	go test -race ./...

# Run integration tests (requires Consul running)
test-integration:
	go test -race -tags=integration ./...

# Start Consul for local integration testing
consul-up:
	docker compose -f docker-compose.test.yml up -d
	@echo "Waiting for Consul to be ready..."
	@until docker compose -f docker-compose.test.yml exec -T consul consul members >/dev/null 2>&1; do \
		sleep 1; \
	done
	@echo "Consul is ready"

# Stop Consul
consul-down:
	docker compose -f docker-compose.test.yml down -v

# Run integration tests with Consul lifecycle management (for CI)
test-integration-ci: consul-up
	@echo "Running integration tests..."
	go test -race -tags=integration ./... || (docker compose -f docker-compose.test.yml down -v && exit 1)
	docker compose -f docker-compose.test.yml down -v

# Run all tests including integration (local development)
test-all: consul-up
	go test -race -tags=integration ./...
	docker compose -f docker-compose.test.yml down -v

# Coverage with integration tests
coverage:
	go test -coverpkg=./... ./... -race -coverprofile=coverage.out -covermode=atomic

coverage-integration: consul-up
	go test -coverpkg=./... -tags=integration ./... -race -coverprofile=coverage.out -covermode=atomic
	docker compose -f docker-compose.test.yml down -v

# Clean up
clean:
	rm -f tagit coverage.out
	docker compose -f docker-compose.test.yml down -v 2>/dev/null || true
