.PHONY: run test tidy build

VERSION := $(shell git describe --tags --dirty --always)
COMMIT  := $(shell git rev-parse --short HEAD)
DATE    := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

JGDBSRC    := cmd/jackgodb/jackgodb.go
JGDBBIN    := bin/jackgodb
LIBSRC    := $(shell find ./internal  -type f -name '*.go')
MODSRC    := ./go.mod ./go.sum

all: jgdbbuild

jgdbbuild: $(JGDBBIN)

$(JGDBBIN): $(MODSRC) $(JGDBSRC) $(LIBSRC)
	go build -o $(JGDBBIN) $(JGDBSRC)

test:
	go test .

cover:
	go test . -coverprofile=cp.out; go tool cover -func=cp.out; go tool cover -html=cp.out -o cp.html; open cp.html

tidy:
	go mod tidy; go mod vendor

jgdb: jgdbbuild
	$(JGDBBIN)
