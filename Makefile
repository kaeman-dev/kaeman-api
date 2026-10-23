BINARY_NAME ?= kaeman-public-api

.PHONY: lint lint-fix test build run clean models security ci release dry-release

lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

test:
	go test $(if $(filter 1,$(shell go env CGO_ENABLED)),-race) ./...

build:
	go build -trimpath -ldflags "-s -w" -o $(BINARY_NAME) ./cli

run: build
	./$(BINARY_NAME)

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe

models:
	@atlas schema inspect --config storage/atlas/atlas.hcl --env generate --url env://src > storage/models_gen.go && gofmt -w storage/models_gen.go

security:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

ci: lint test security

release:
	goreleaser release --clean

dry-release:
	goreleaser release --snapshot --clean

-include Makefile.local
