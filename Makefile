BINARY := jira-tui
PKG    := ./cmd/jira-tui

.PHONY: build run test vet lint fmt snapshot release clean

build:
	go build -o bin/$(BINARY) $(PKG)

run:
	go run $(PKG)

test:
	go test -race ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

# Local release dry-run: builds artifacts into dist/ without publishing.
snapshot:
	goreleaser release --snapshot --clean

# Publish a release (run from a tagged commit; needs GITHUB_TOKEN + HOMEBREW_TAP_TOKEN).
release:
	goreleaser release --clean

clean:
	rm -rf bin dist
