##
## Copyright 2025 PT. Fajar Mitra Indah. All rights reserved.
## Use of this source code is governed by a MIT style
## license that can be found in the LICENSE file.
##

PACKAGE = site-dev-backend
COMMIT_HASH = `git rev-parse --short HEAD 2>/dev/null`
BUILD_DATE = `date +%FT%T%z`
LDFLAGS = -ldflags "-X ${PACKAGE}/iqlib.CommitHash=${COMMIT_HASH} -X ${PACKAGE}/iqlib.BuildDate=${BUILD_DATE}"
NOGI_LDFLAGS = -ldflags "-X ${PACKAGE}/iqlib.BuildDate=${BUILD_DATE}"

DIR_SOURCE = $(shell find . -maxdepth 10 -type f -not -path '*/vendor*' -name '*.go' | xargs -I {} dirname {} | sort | uniq)


.PHONY: test fmt lint vet ineffassign format test test-race test-cover help
.DEFAULT_GOAL := help

build: ## Build binary
	go build ${LDFLAGS} ${PACKAGE}

fmt: ## Check gofmt linter
	for dir in ${DIR_SOURCE}; do \
		if [ "`gofmt -l $$dir | grep -v vendor/ | tee /dev/stderr`" ]; then \
			echo "^ improperly formatted go files" && echo && exit 1; \
		fi \
	done

vendoring: ## Create vendor directory
	rm -rf vendor
	go mod vendor

lint: ## Check golint linter
	for dir in ${DIR_SOURCE}; do \
		if [ "`golint $$dir | grep -v vendor/ | tee /dev/stderr`" ]; then \
			echo "^ golint errors!" && echo && exit 1; \
		fi \
	done

vet: ## Check go vet linter
	for dir in ${DIR_SOURCE}; do \
		if [ "`go vet $$dir | grep -v vendor/ | tee /dev/stderr`" ]; then \
			echo "^ go vet errors!" && echo && exit 1; \
		fi \
	done

ineffassign: ## Check ineffectual assignments source code
	for dir in ${DIR_SOURCE}; do \
		if [ "`ineffassign $$dir | grep -v vendor/ | tee /dev/stderr`" ]; then \
			echo "^ ineffectual assignment detected!" && echo && exit 1; \
		fi \
	done

format: ## Run formating source code
	for dir in ${DIR_SOURCE}; do \
		gofmt -l -w $$dir | grep -v vendor/ | tee /dev/stderr; \
		golint $$dir | grep -v vendor/ | tee /dev/stderr; \
		goimports -l -w $$dir | grep -v vendor/ | tee /dev/stderr; \
		ineffassign $$dir | grep -v vendor/ | tee /dev/stderr; \
	done

test: ## Run tests
	rm -f profile.cov test.tmp; \
	echo "mode: count" >> profile.cov
	for dir in ${DIR_SOURCE}; do \
		gofmt -l -w -s $$dir/*.go | tee /dev/stderr; \
		golint $$dir | tee /dev/stderr; \
		ineffassign $$dir | grep -v vendor/ | tee /dev/stderr; \
		go vet $$dir | tee /dev/stderr; \
		go test -short -covermode=count -coverprofile=$$dir/profile.tmp $$dir | tee -a test.tmp; \
		if [ "`grep FAIL test.tmp`" ]; then \
			exit 1; \
		fi; \
		if [ -f $$dir/profile.tmp ]; then \
			cat $$dir/profile.tmp | tail -n +2 >> profile.cov; \
			rm $$dir/profile.tmp; \
		fi; \
	done
	go tool cover -func profile.cov;
	rm -f profile.cov test.tmp; \

test-race: ## Run tests with race detector
	env GORACE="halt_on_error=1" go test -short -race ${DIR_SOURCE}

test-cover: ## Generate test coverage report
	rm -f profile.cov test.tmp; \
	echo "mode: count" >> profile.cov
	for dir in ${DIR_SOURCE}; do \
	    go test -short -covermode=count -coverprofile=$$dir/profile.tmp $$dir | tee -a test.tmp; \
		if [ "`grep FAIL test.tmp`" ]; then \
			exit 1; \
		fi; \
	    if [ -f $$dir/profile.tmp ]; then \
	        cat $$dir/profile.tmp | tail -n +2 >> profile.cov; \
	        rm $$dir/profile.tmp; \
	    fi; \
	done
	go tool cover -func profile.cov;
	rm -f profile.cov test.tmp; \

help:
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
