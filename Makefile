.PHONY: test vet fmt cover demo all

all: fmt vet test

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Show the framework's own reporters in action.
demo:
	go test ./examples/ -v
