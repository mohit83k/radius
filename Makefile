# File: Makefile

APP_NAME=radius-controlplane
LOGGER_NAME=redis-controlplane-logger

# Build binaries
build:
	go build -o bin/$(APP_NAME) ./cmd/radius-controlplane
	go build -o bin/$(LOGGER_NAME) ./cmd/redis-controlplane-logger

# Run tests
test:
	go test ./internal/... -v

# Run go mod tidy
tidy:
	go mod tidy

# Clean up builds
clean:
	rm -rf bin
	go clean -a

# Docker up/down
up:
	docker compose -f docker/docker-compose.yml up --build

down:
	docker compose -f docker/docker-compose.yml down

restart:
	docker compose -f docker/docker-compose.yml restart

logs:
	docker compose -f docker/docker-compose.yml logs -f

logs-controlplane:
	docker compose -f docker/docker-compose.yml logs -f radius-controlplane


radclient-bash:
	docker compose -f docker/docker-compose.yml exec radclient-test bash

ps:
	docker compose -f docker/docker-compose.yml ps


integration-test-build:
	make clean 
	docker compose -f docker/docker-compose.yml up --build -d
	make integration-test

# Run integration test inside radclient-test container
integration-test-up:
	docker compose -f docker/docker-compose.yml up -d
	@echo "⏳ Waiting for radius-controlplane to be ready..."
	@sleep 2
	@echo "🚀 Running integration test script..."
	@make run-test-script
	@echo "🔍 Checking logs..."
	@make check-test-script-logs
	@echo "Test completed. (Expecting 9 logs if 3 users × 3 types)"

integration-test:
	@make rotate-logs
	@make restart
	@echo "Waiting for radius-controlplane to be ready..."
	@sleep 2
	@echo "Running integration test script..."
	@make run-test-script
	@echo " Checking logs..."
	@make check-test-script-logs
	@echo "Test completed. (Expecting 9 logs if 3 users × 3 types)"


run-test-script:
	@docker compose -f docker/docker-compose.yml exec radclient-test ./test_concurrent_requests.sh

check-test-script-logs:
	@grep '"Stored accounting record"' ./docker/persisted_logs/radius_server.log \
		| sort | uniq | tee last_test.log | wc -l

	@rm last_test.log

clean-logs:
	rm -f ./docker/persisted_logs/*
	

rotate-logs:
	@NOW=$$(date +"%Y%m%d_%H%M%S"); \
	for FILE in docker/persisted_logs/radius_server.log docker/persisted_logs/radius_logger.log; do \
	  if [ -f $$FILE ]; then \
	    BASENAME=$$(basename $$FILE .log); \
	    mv $$FILE docker/persisted_logs/$$BASENAME_$$NOW.log; \
	    echo "🔁 Rotated $$FILE to $$BASENAME_$$NOW.log"; \
	  else \
	    echo "ℹ️  No log file $$FILE to rotate"; \
	  fi; \
	  touch $$FILE; \
	done
	@make restart-logger


restart-logger:
	docker compose -f docker/docker-compose.yml restart redis-controlplane-logger

