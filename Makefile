

PACKAGES=$(shell go list ./... | grep -v '/vendor/')

VETARGS?=-asmdecl -atomic -bool -buildtags -copylocks -methods \
         -nilfunc -printf -rangeloops -shift -structtags -unsafeptr

build: test
	@echo "--> Running go build"
	@go generate $(PACKAGES)
	@go build

test:
	@echo "--> Running go test"
	@go test $(PACKAGES) $(TESTARGS) -timeout=30s -parallel=4


cover:
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


.PHONY: cov test cover format vet
