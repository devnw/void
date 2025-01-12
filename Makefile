#-------------------------------------------------------------------------
# Variables
# ------------------------------------------------------------------------
env=CGO_ENABLED=1
op= op run --env-file="./.env" -- 
SHELL := $(shell which bash)
fuzzsh=https://raw.githubusercontent.com/devnw/workflows/refs/heads/main/fuzz.sh

#-------------------------------------------------------------------------
# Custom Targets
# ------------------------------------------------------------------------
release-dev:
	$(env) $(op) goreleaser release --clean --snapshot

#-------------------------------------------------------------------------
# Standard Targets
# ------------------------------------------------------------------------
pre-commit: fmt update upgrade tidy lint build test

gomod2nix:
	gomod2nix generate

detect-secrets-baseline:
	detect-secrets scan > .secrets.baseline

lint:
	golangci-lint run

build: gomod2nix
	$(env) go build ./...

upgrade:
	pre-commit autoupdate
	go get -u ./...

update:
	git submodule update --recursive

test: 
	CGO_ENABLED=1 go test -v -cover -failfast -race ./...

fuzz:
	curl -fsSL $(fuzzsh) | $(SHELL)

bench:
	go test -bench=. -benchmem ./...

fmt: 
	prettier --config ./.pretterrc.yaml --write .
	find . -type f -name "*.nix" -exec nixfmt {} +
	goimports -w .
	gofmt -s -w .

tidy: fmt
	go mod tidy

release: 
	if [ -z "$(tag)" ]; then echo "tag is required"; exit 1; fi
	git tag -a ${tag} -m "${tag}"
	git push origin ${tag}

clean: 
	rm -rf dist
	rm -rf out

#-------------------------------------------------------------------------
# CI targets
#-------------------------------------------------------------------------
build-ci: lint
	$(env) go build ./...

test-ci: build-ci 
	CGO_ENABLED=1 go test \
				-cover \
				-covermode=atomic \
				-coverprofile=coverage.txt \
				-failfast \
				-race ./...
	make fuzz


bench-ci: test-ci
	go test -bench=. ./... | tee bench-output.txt


release-ci: bench-ci
	$(env) $(op) goreleaser release --clean

#-------------------------------------------------------------------------
# Force targets
#-------------------------------------------------------------------------

FORCE: 

#-------------------------------------------------------------------------
# Phony targets
#-------------------------------------------------------------------------

.PHONY: build test lint fuzz all clean FORCE bench fmt tidy release
