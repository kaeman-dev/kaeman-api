BINARY_NAME=kaeman-public-api
LDFLAGS=-s -w
GOFLAGS=-trimpath

.PHONY: build run clean models

build:
	@printf 'Building $(BINARY_NAME) (all features)...\n'
	@go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) ./cli

run: build
	./$(BINARY_NAME)

clean:
	rm -f $(BINARY_NAME)

models:
	@atlas schema inspect --config storage/atlas/atlas.hcl --env generate --url env://src > storage/models_gen.go && gofmt -w storage/models_gen.go

-include Makefile.local
