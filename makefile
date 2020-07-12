.PHONY: test
test:
	go test -race -cover ./...

.PHONY: cover
cover:
	go test -race $$(go list ./...) -v -coverprofile=coverage.txt
	go tool cover -func=coverage.txt

.PHONY: dev
dev:
	go run ./cmd/main.go

.PHONY: build
build:
	go build -o bin/app ./cmd/main.go 

.PHONY: run
run: build
	./bin/app
