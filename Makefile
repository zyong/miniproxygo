# init project path
NAME=proxy
WORKROOT := $(shell pwd)
OUTDIR   := $(WORKROOT)/output
OS		 := $(shell go env GOOS)

# init environment variables
export PATH        := $(shell go env GOPATH)/bin:$(PATH)
export GO111MODULE := on

# init command params
GO           := go
GOBUILD      := CGO_ENABLED=0 $(GO) build
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

# make, make all
all: prepare compile package

# make, make strip
strip: prepare compile-strip package

# make compile, go build
compile: test linux-amd64 linux-arm64 macos-amd64 macos-arm64
linux-amd64:
	GOARCH=amd64 GOOS=linux $(GOBUILD) -ldflags "-w -s -X main.version=$(M_VERSION) -X main.commit=$(GIT_COMMIT)" -o $(OUTDIR)/bin/$(NAME)-$@

linux-arm64:
	GOARCH=arm64 GOOS=linux $(GOBUILD) -ldflags "-w -s -X main.version=$(M_VERSION) -X main.commit=$(GIT_COMMIT)" -o $(OUTDIR)/bin/$(NAME)-$@

macos-amd64:
	GOARCH=amd64 GOOS=linux $(GOBUILD) -ldflags "-w -s -X main.version=$(M_VERSION) -X main.commit=$(GIT_COMMIT)" -o $(OUTDIR)/bin/$(NAME)-$@

macos-arm64:
	GOARCH=arm64 GOOS=linux $(GOBUILD) -ldflags "-w -s -X main.version=$(M_VERSION) -X main.commit=$(GIT_COMMIT)" -o $(OUTDIR)/bin/$(NAME)-$@


# make package
package:
	cp -r conf $(OUTDIR)

# make deps
deps:
	$(call PIP_INSTALL_PKG, pre-commit)
	$(call INSTALL_PKG, staticcheck, honnef.co/go/tools/cmd/staticcheck)
	$(call INSTALL_PKG, license-eye, github.com/apache/skywalking-eyes/cmd/license-eye@latest)

# make precommit, enable autoupdate and install with hooks
precommit:
	pre-commit autoupdate
	pre-commit install --install-hooks

# make check
check:
	$(GO) get honnef.co/go/tools/cmd/staticcheck
	$(STATICCHECK) ./...

# make clean
clean:
	$(GOCLEAN)
	rm -rf $(OUTDIR)
	rm -rf $(GOPATH)/pkg/linux_amd64

# avoid filename conflict and speed up build
.PHONY: all prepare compile test package clean build
