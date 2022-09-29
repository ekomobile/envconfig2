SHELL:=/bin/bash

.PHONY: all
all:

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test ./...
