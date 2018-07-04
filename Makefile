#!/usr/bin/env bash
# Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

GOTOOLS = github.com/ystia/gocovermerge github.com/kardianos/govendor github.com/jteeuwen/go-bindata/... github.com/abice/go-enum github.com/google/addlicense

VETARGS?=-all -asmdecl -atomic -bool -buildtags -copylocks -methods \
         -nilfunc -printf -rangeloops -shift -structtags -unsafeptr

VERSION=$(shell grep "yorc_version" versions.yaml | awk '{print $$2}')
COMMIT_HASH=$(shell git rev-parse HEAD)

buildnformat: build header format

build: test
	@echo "--> Running go build"
	@CGO_ENABLED=0 go build $(BUILD_ARGS) -ldflags '-X github.com/ystia/yorc/commands.version=v$(VERSION) -X github.com/ystia/yorc/commands.gitCommit=$(COMMIT_HASH)'

generate: checks
	@go generate ./...

checks:
	@./build/checks.sh $(GOTOOLS)

header:
	@echo "--> Adding licensing headers if necessary"
	@./build/header.sh

dist: buildnformat
	@rm -rf ./dist && mkdir -p ./dist
	@echo "--> Creating an archive"
	@tar czvf yorc.tgz yorc && echo "TODO: clean this part after CI update" &&  cp yorc yorc.tgz dist/
	@cd doc && make html latexpdf && cd _build && cp -r html latex/Yorc.pdf ../../dist
	@cd ./dist && zip -r yorc-server-$(VERSION)-documentation.zip html Yorc.pdf && zip yorc-server-$(VERSION)-distrib.zip yorc yorc-server-$(VERSION)-documentation.zip

test: generate
ifndef SKIP_TESTS
	@echo "--> Running go test"
	@export PATH=$$PWD/build:$$PATH; go test $(TESTARGS) -p 1 ./...
endif


cover: 
	@go test -p 1 -cover $(COVERARGS) ./...  

format:
	@echo "--> Running go fmt"
	@go fmt ./...

vet:
	@echo "--> Running go tool vet $(VETARGS) ."
	@go list ./... \
		| cut -d '/' -f 4- \
		| xargs -n1 \
			go tool vet $(VETARGS) ;\
	if [ $$? -ne 0 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for reviewal."; \
	fi

tools:
	@./build/tools.sh $(GOTOOLS)

savedeps: checks
	@godep save -v ./...

restoredeps: checks
	@godep restore -v

.PHONY: buildnformat build cov checks test cover format vet tools dist
