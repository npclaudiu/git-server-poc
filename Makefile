EXECUTABLE=git-server-poc

.PHONY: all
all: debug release

.PHONY: debug
debug:
	go build -v -o bin/$(EXECUTABLE) \
		./cmd/$(EXECUTABLE)/main.go

.PHONY: release
release:
	go build -v -o bin/$(EXECUTABLE) \
		-ldflags="-s -w" \
		-trimpath \
		./cmd/$(EXECUTABLE)/main.go

.PHONY: clean
clean:
	rm -rf bin

.PHONY: test
test:
	go test ./...

.PHONY: smoke-test
smoke-test:
	go test -v -count=1 ./tests/smoke/...

.PHONY: format
format:
	go fmt ./...

.PHONY: lint
lint:
	go vet ./...

.PHONY: tidy
tidy:
	go mod tidy
	go mod verify

.PHONY: devenv-up
devenv-up:
	pnpm --prefix ./devenv devenv up

.PHONY: devenv-down
devenv-down:
	pnpm --prefix ./devenv devenv down

.PHONY: ms-migrate
ms-migrate:
	pnpm --prefix ./devenv devenv metastore migrate

.PHONY: ms-gen
ms-gen:
	pnpm --prefix ./devenv devenv metastore generate
