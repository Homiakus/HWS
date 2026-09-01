.PHONY: fmt test test-race vet check demo

fmt:
	gofmt -w $$(find . -name '*.go' -type f)

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

check: test vet

demo:
	go run ./cmd/hwsctl
