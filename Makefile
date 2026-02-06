.PHONY: build run dev test lint clean

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w \
	-X github.com/devbydaniel/meetingcli/internal/version.Version=$(VERSION) \
	-X github.com/devbydaniel/meetingcli/internal/version.Commit=$(COMMIT) \
	-X github.com/devbydaniel/meetingcli/internal/version.Date=$(DATE)

build:
	go build -ldflags '$(LDFLAGS)' -o meeting ./cmd/meeting

run: build
	./meeting

dev:
	MEETINGCLI_MEETINGS_DIR=./dev-meetings go run -ldflags '$(LDFLAGS)' ./cmd/meeting

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -f meeting
