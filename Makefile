install:
	@go build -o bin/mx ./cmd/
	@cp bin/mx $(shell go env GOPATH)/bin/mx