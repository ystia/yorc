GOTOOLS = golang.org/x/tools/cmd/stringer github.com/tools/godep github.com/jteeuwen/go-bindata

PACKAGES=$(shell go list ./... | grep -v '/vendor/')
PACKAGES_MINUS_TASKS=$(shell go list ./... | grep -v '/vendor/' | grep -v 'tasks')

VETARGS?=-asmdecl -atomic -bool -buildtags -copylocks -methods \
         -nilfunc -printf -rangeloops -shift -structtags -unsafeptr

build: test
	@echo "--> Running go build"
	@go generate $(PACKAGES)
	@CGO_ENABLED=0 go build

checks:
	@./build/checks.sh $(GOTOOLS)

dist: build
	@echo "--> Creating an archive"
	@tar czvf janus.tgz janus

test: checks
	@echo "--> Running go test"
	@go test $(PACKAGES_MINUS_TASKS) $(TESTARGS) -timeout=30s -parallel=0
	@go test ./tasks/... $(TESTARGS) -timeout=30s -parallel=0


cover: build
	go list ./... | xargs -n1 go test --cover

format:
	@echo "--> Running go fmt"
	@go fmt $(PACKAGES)

vet:
	@echo "--> Running go tool vet $(VETARGS) ."
	@go list ./... \
		| grep -v '/vendor/' \
		| cut -d '/' -f 4- \
		| xargs -n1 \
			go tool vet $(VETARGS) ;\
	if [ $$? -ne 0 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for reviewal."; \
	fi

tools:
	go get -u -v $(GOTOOLS)

.PHONY: cov test cover format vet tools
