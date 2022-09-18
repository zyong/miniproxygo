# init project path
WORKROOT := $(shell pwd)
OUTDIR   := $(WORKROOT)/output
OS		 := $(shell go env GOOS)

# init environment variables
export PATH        := $(shell go env GOPATH)/bin:$(PATH)
export GO111MODULE := on

# init command params
GO           := go
GOBUILD      := $(GO) build
GOTEST       := $(GO) test
GOVET        := $(GO) vet
GOGET        := $(GO) get
GOGEN        := $(GO) generate
GOCLEAN      := $(GO) clean
GOINSTALL    := $(GO) install
GOFLAGS      := -race
STATICCHECK  := staticcheck
LICENSEEYE   := license-eye
PIP          := pip3
PIPINSTALL   := $(PIP) install

# init arch
ARCH := $(shell getconf LONG_BIT)
ifeq ($(ARCH),64)
	GOTEST += $(GOFLAGS)
endif

# init miniproxy version
M_VERSION ?= $(shell cat VERSION)
# init git commit id
GIT_COMMIT ?= $(shell git rev-parse HEAD)

# init miniproxy packages
M_PKGS := $(shell go list ./...)

# go install package
# $(1) package name
# $(2) package address
define INSTALL_PKG
	@echo installing $(1)
	$(GOINSTALL) $(2)
	@echo $(1) installed
endef

define PIP_INSTALL_PKG
	@echo installing $(1)
	$(PIPINSTALL) $(1)
	@echo $(1) installed
endef

# make, make all
all: prepare compile package

# make, make strip
strip: prepare compile-strip package

# make prepare, download dependencies
prepare: prepare-dep prepare-gen
prepare-dep:
	$(call INSTALL_PKG, goyacc, golang.org/x/tools/cmd/goyacc)

# make compile, go build
compile: test build
build:
ifeq ($(OS),darwin)
	$(GOBUILD) -ldflags "-X main.version=$(M_VERSION) -X main.commit=$(GIT_COMMIT)"
else
	$(GOBUILD) -ldflags "-X main.version=$(M_VERSION) -X main.commit=$(GIT_COMMIT) -extldflags=-static"
endif

# make compile-strip, go build without symbols and DWARFs
compile-strip: test build-strip
build-strip:
ifeq ($(OS),darwin)
	$(GOBUILD) -ldflags "-X main.version=$(M_VERSION) -X main.commit=$(GIT_COMMIT) -s -w"
else
	$(GOBUILD) -ldflags "-X main.version=$(M_VERSION) -X main.commit=$(GIT_COMMIT) -extldflags=-static -s -w"
endif

# make test, test your code
test: test-case vet-case
test-case:
	$(GOTEST) -cover ./...
vet-case:
	${GOVET} ./...

# make coverage for codecov
coverage:
	echo -n > coverage.txt
	for pkg in $(M_PKGS) ; do $(GOTEST) -coverprofile=profile.out -covermode=atomic $${pkg} && cat profile.out >> coverage.txt; done

# make package
package:
	mkdir -p $(OUTDIR)/bin
	mv miniproxy  $(OUTDIR)/bin
	cp -r conf $(OUTDIR)

# make deps
deps:
	$(call PIP_INSTALL_PKG, pre-commit)
	$(call INSTALL_PKG, goyacc, golang.org/x/tools/cmd/goyacc)
	$(call INSTALL_PKG, staticcheck, honnef.co/go/tools/cmd/staticcheck)
	$(call INSTALL_PKG, license-eye, github.com/apache/skywalking-eyes/cmd/license-eye@latest)

# make precommit, enable autoupdate and install with hooks
precommit:
	pre-commit autoupdate
	pre-commit install --install-hooks

# make check
check:
	$(STATICCHECK) ./...

# make license-check, check code file's license declaration
license-check:
	$(LICENSEEYE) header check

# make license-fix, fix code file's license declaration
license-fix:
	$(LICENSEEYE) header fix

# make clean
clean:
	$(GOCLEAN)
	rm -rf $(OUTDIR)
	rm -rf $(GOPATH)/pkg/linux_amd64

# avoid filename conflict and speed up build
.PHONY: all prepare compile test package clean build
