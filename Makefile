SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.ONESHELL:
.DEFAULT_GOAL := help

.PHONY: help fmt fmt-check build check

help:
	@printf '%s\n' \
	  'make fmt        Format Go source files' \
	  'make fmt-check  Check Go formatting without changing files' \
	  'make build      Build all Go packages' \
	  'make check      Check formatting and build'

fmt:
	@gofmt -w .

fmt-check:
	@mapfile -d '' files < <(git ls-files --cached --others --exclude-standard -z -- '*.go' ':!:vendor/**')
	if (( $${#files[@]} == 0 )); then
	  echo 'No Go files yet; skipping formatting check.'
	  exit 0
	fi
	unformatted=$$(gofmt -l "$${files[@]}")
	if [[ -n "$$unformatted" ]]; then
	  printf 'Run gofmt on the following files:\n%s\n' "$$unformatted"
	  exit 1
	fi

build:
	@if [[ ! -f go.mod ]]; then
	  if [[ -n "$$(git ls-files --cached --others --exclude-standard -- '*.go' ':!:vendor/**')" ]]; then
	    echo 'Go files exist but go.mod is missing. Initialize the Go module first.'
	    exit 1
	  fi
	  echo 'No Go module yet; skipping build.'
	  exit 0
	fi
	go build ./...

check: fmt-check build
