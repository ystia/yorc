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

GOTOOLS = github.com/abice/go-enum github.com/google/addlicense

VETARGS?=-all -asmdecl -atomic -bool -buildtags -copylocks -methods \
         -nilfunc -printf -rangeloops -shift -structtags -unsafeptr

VERSION:=$(shell grep "yorc_version" versions.yaml | awk '{print $$2}')
VERSION_META:=$(shell echo `echo $(BUILD_TAGS) | tr ' ' '.'`)
VERSION:=$(if $(VERSION_META),$(VERSION)+$(VERSION_META),$(VERSION))
COMMIT_HASH=$(shell git rev-parse HEAD)
ANSIBLE_VERSION=$(shell grep "ansible_version" versions.yaml | awk '{print $$2}')
CONSUL_VERSION=$(shell grep "consul_version" versions.yaml | awk '{print $$2}')
ALIEN4CLOUD_VERSION=$(shell grep "alien4cloud_version" versions.yaml | awk '{print $$2}')
TERRAFORM_VERSION=$(shell grep "terraform_version" versions.yaml | awk '{print $$2}')
TF_CONSUL_PLUGIN_VERSION=$(shell grep "tf_consul_plugin_version" versions.yaml | awk '{print $$2}')
TF_AWS_PLUGIN_VERSION=$(shell grep "tf_aws_plugin_version" versions.yaml | awk '{print $$2}')
TF_OPENSTACK_PLUGIN_VERSION=$(shell grep "tf_openstack_plugin_version" versions.yaml | awk '{print $$2}')
TF_GOOGLE_PLUGIN_VERSION=$(shell grep "tf_google_plugin_version" versions.yaml | awk '{print $$2}')
YORC_VERSION=$(shell grep "yorc_version" versions.yaml | awk '{print $$2}')

# Should be updated when changing major version
YORC_PACKAGE=github.com/ystia/yorc/v3

export GO111MODULE=on
export BUILD_DIR=$(shell pwd)/build
export GOBIN=$(BUILD_DIR)/bin
export PATH=$(GOBIN):$(shell echo $$PATH)
export CGO_ENABLED=0

build: test
	@echo "--> Running go build"
	@go build -o yorc -tags "$(BUILD_TAGS)" $(BUILD_ARGS) -ldflags "-X $(YORC_PACKAGE)/commands.version=v$(VERSION) -X $(YORC_PACKAGE)/commands.gitCommit=$(COMMIT_HASH) \
	 -X $(YORC_PACKAGE)/commands.TfConsulPluginVersion=$(TF_CONSUL_PLUGIN_VERSION) \
	 -X $(YORC_PACKAGE)/commands.TfAWSPluginVersion=$(TF_AWS_PLUGIN_VERSION) \
	 -X $(YORC_PACKAGE)/commands.TfOpenStackPluginVersion=$(TF_OPENSTACK_PLUGIN_VERSION) \
	 -X $(YORC_PACKAGE)/commands.TfGooglePluginVersion=$(TF_GOOGLE_PLUGIN_VERSION) \
	 -X $(YORC_PACKAGE)/commands/bootstrap.ansibleVersion=$(ANSIBLE_VERSION) \
	 -X $(YORC_PACKAGE)/commands/bootstrap.consulVersion=$(CONSUL_VERSION) \
	 -X $(YORC_PACKAGE)/commands/bootstrap.alien4cloudVersion=$(ALIEN4CLOUD_VERSION) \
	 -X $(YORC_PACKAGE)/commands/bootstrap.terraformVersion=$(TERRAFORM_VERSION) \
	 -X $(YORC_PACKAGE)/commands/bootstrap.yorcVersion=$(YORC_VERSION)"
	 @rm -f ./build/embeddedResources.zip
	 @cd ./commands && zip -q -r ../build/embeddedResources.zip ./bootstrap/resources/*
	 @cd ./data && zip -q -r ../build/embeddedResources.zip ./*
	 @cat ./build/embeddedResources.zip >> ./yorc
	 @zip -A ./yorc > /dev/null

generate: checks
	@go generate ./...

checks:
	@./build/checks.sh $(GOTOOLS)

header:
	@echo "--> Adding licensing headers if necessary"
	@./build/header.sh

dist: build
	@rm -rf ./dist && mkdir -p ./dist
	@echo "--> Creating an archive"
	@tar czvf "yorc-$(VERSION).tgz" yorc &&  cp yorc "yorc-$(VERSION)".tgz dist/
	@cd doc && make html latexpdf && cd _build && cp -r html latex/Yorc.pdf ../../dist
	@cd ./dist && zip -r "yorc-server-$(VERSION)-documentation.zip" html Yorc.pdf && zip "yorc-server-$(VERSION)-distrib.zip" yorc "yorc-server-$(VERSION)-documentation.zip"

test: generate header format
ifndef SKIP_TESTS
	@echo "--> Running go test"
	@go test -tags "testing $(BUILD_TAGS)" $(TESTARGS) -p 1 ./...
endif

json-test: generate header format
	@echo "--> Running go test with json output"
# -count is for disabling test cache
	@go test -tags "testing $(BUILD_TAGS)" $(TESTARGS) -json -count=1 -p 1 ./... > tests-reports.json

cover:
	@go test -tags "testing $(BUILD_TAGS)"  -p 1 -cover $(COVERARGS) ./...

format:
	@echo "--> Running go fmt"
	@go fmt ./...

vet:
	@echo "--> Running go tool vet $(VETARGS) ."
	@go vet $(VETARGS) ./...

tools:
	@echo "--> Installing $(GOTOOLS)"
	@go install $(GOTOOLS)

.PHONY: build cov checks test cover format vet tools dist
