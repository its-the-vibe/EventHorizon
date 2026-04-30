BINARY   := eventhorizon
PKG      := ./...
COVER    := coverage.out

.PHONY: build test lint clean

## build: compile the binary
build:
	go build -o $(BINARY) .

## test: run all tests with the race detector and produce a coverage profile
test:
	go test -race -coverprofile=$(COVER) $(PKG)

## lint: run go vet across all packages
lint:
	go vet $(PKG)

## clean: remove build and test artifacts
clean:
	rm -f $(BINARY) $(COVER)
