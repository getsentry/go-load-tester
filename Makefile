build: lint
	go mod download && go build
.PHONY: build

make-docs:
	go run . make-docs
.PHONY: make-docs

test:
	go test ./... -v -count=1
.PHONY: test

fmt:
	go fmt .
.PHONY: fmt

lint:
	go vet .
.PHONY: lint
