build: lint
	go mod download && go build
.PHONY: build

update-docs:
	go run . update-docs
.PHONY: update-docs

test:
	go test ./... -v -count=1
.PHONY: test

fmt:
	go fmt .
.PHONY: fmt

lint:
	go vet .
.PHONY: lint
