# ZetaChat Makefile — common development shortcuts.
# Usage:  make build | make test | make vet | ...

BINARY := zetachat

.PHONY: build run test vet fmt tidy install clean

build:
	go build -o $(BINARY) ./cmd/zetachat

run:
	go run ./cmd/zetachat

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

tidy:
	go mod tidy

install:
	go install ./cmd/zetachat

clean:
	go clean
	rm -f $(BINARY)
