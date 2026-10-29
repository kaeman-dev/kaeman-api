BINARY_NAME ?= kaeman-api

.PHONY: lint lint-fix test build run clean models security ci release dry-release

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run

lint-fix:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run --fix

test:
	go test $(if $(filter 1,$(shell go env CGO_ENABLED)),-race) ./...

build:
	go build -trimpath -ldflags "-s -w" -o $(BINARY_NAME) ./cli

run: build
	./$(BINARY_NAME)

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe

models:
	@atlas schema inspect --config shared/storage/atlas/atlas.hcl --env generate --url env://src > shared/storage/models_gen.go && gofmt -w shared/storage/models_gen.go

security:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

ci: lint test security

release:
	goreleaser release --clean

dry-release:
	goreleaser release --snapshot --clean

-include Makefile.local
