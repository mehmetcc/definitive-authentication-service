.PHONY: run test docs build fmt vet lint coverage clean docker-up docker-down help

# run the app
run:
	go run main.go

# run all tests verbosely
test:
	go test -v ./...

# regenerate Swagger docs
docs:
	rm -rf docs
	swag init --parseDependency --parseInternal

# build the binaries
build:
	go build -v ./...

# format code
fmt:
	go fmt ./...

# vet for potential issues
vet:
	go vet ./...

# static analysis (requires staticcheck installed)
lint:
	staticcheck ./...

# run tests with coverage report
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# clean generated artifacts
clean:
	rm -f ./coverage.out

# bring up dependencies with docker-compose
docker-up:
	docker-compose up -d

# tear down dependencies with docker-compose
docker-down:
	docker-compose down

# show available targets
help:
	@echo "Usage: make [target]"
	@echo "Available targets:" \
		"run test docs build fmt vet lint coverage clean docker-up docker-down help"
