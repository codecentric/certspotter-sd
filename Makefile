MODULE   = $(shell env GO111MODULE=on $(GO) list -m)
DATE    ?= $(shell date -u +%F)
VERSION ?= $(shell git describe --tags --abbrev=0 --match=v* 2> /dev/null)
PKGS     = $(or $(PKG),$(shell env GO111MODULE=on $(GO) list ./...))
TESTPKGS = $(shell env GO111MODULE=on $(GO) list -f \
			'{{ if or .TestGoFiles .XTestGoFiles }}{{ .ImportPath }}{{ end }}' \
			$(PKGS))
BIN = $(CURDIR)/bin
OUT = $(CURDIR)/out

GO      = go
TIMEOUT = 15
LDFLAGS = "-X $(MODULE)/internal/version.Version=$(VERSION) -X $(MODULE)/internal/version.BuildDate=$(DATE)"

PREV_VERSION = $(shell git describe --abbrev=0 --match=v* $(VERSION)^ 2> /dev/null)

DEBUG = 0
Q = $(if $(filter 1,$DEBUG),,@)
M = $(shell printf ">_ ")

export GO111MODULE=on

.DEFAULT_GOAL:=all
.PHONY: all
all: clean fmt lint static test build 

# Tools

$(BIN):
	@mkdir -p $@
$(BIN)/%: | $(BIN) ; $(info $(M) building $(PACKAGE)...)
	$Q tmp=$$(mktemp -d); \
	   env GO111MODULE=off GOPATH=$$tmp GOBIN=$(BIN) $(GO) get $(PACKAGE) \
		|| ret=$$?; \
	   rm -rf $$tmp ; exit $$ret

GOIMPORTS = $(BIN)/goimports
$(BIN)/goimports: PACKAGE=golang.org/x/tools/cmd/goimports

GOLINT = $(BIN)/golint
$(BIN)/golint: PACKAGE=golang.org/x/lint/golint

GOSTATICCHECK = $(BIN)/staticcheck
$(BIN)/staticcheck: PACKAGE=honnef.co/go/tools/cmd/staticcheck

GOX = $(BIN)/gox
$(BIN)/gox: PACKAGE=github.com/mitchellh/gox

# Tests

TEST_TARGETS := test-default test-bench test-short test-verbose test-race
.PHONY: $(TEST_TARGETS) test
test-bench:   ARGS=-run=__absolutelynothing__ -bench=. ## run test benchmarks
test-short:   ARGS=-short        ## run only short tests
test-verbose: ARGS=-v            ## run tests with verbose
test-race:    ARGS=-race         ## run tests with race detector
$(TEST_TARGETS): NAME=$(MAKECMDGOALS:test-%=%)
$(TEST_TARGETS): test
test: fmt lint static ; $(info $(M) running $(NAME:%=% )tests...) @ ## run tests
	$Q $(GO) test -timeout $(TIMEOUT)s $(ARGS) $(TESTPKGS)

.PHONY: static
static: fmt lint | $(GOSTATICCHECK) ; $(info $(M) running staticcheck...) @ ## run staticcheck
	$Q $(GOSTATICCHECK) $(PKGS) 

.PHONY: lint
lint: | $(GOLINT) ; $(info $(M) running golint...) @ ## run golint
	$Q $(GOLINT) -set_exit_status $(PKGS)

.PHONY: fmt
fmt: | $(GOIMPORTS) ; $(info $(M) running goimports...) @ ## run goimports
	$Q $(GOIMPORTS) -l -w -local $(MODULE) $(subst $(MODULE),.,$(PKGS))

# Build

.PHONY: build
build: fmt lint static ; $(info $(M) building local executable...) @ ## build local executable
	$Q $(GO) build \
		-ldflags $(LDFLAGS) \
		-o $(OUT)/bin/certspotter-sd \
		$(MODULE)/cmd/certspotter-sd

.PHONY: build-release
build-release: fmt lint static | $(GOX) ; $(info $(M) building all executables...) @ ## build release executables
	$Q $(GOX) \
		-osarch "freebsd/amd64 freebsd/arm linux/amd64 linux/arm linux/arm64" \
		-ldflags $(LDFLAGS) \
		-output "$(OUT)/bin/{{.Dir}}.{{.OS}}-{{.Arch}}" \
		$(MODULE)/cmd/certspotter-sd

# Release

.PHONY: release
release: fmt lint static test build-release changelog ; $(info $(M) creating releases...) @ ## creating release archives
	$Q cd $(OUT); for file in ./bin/certspotter-sd.* ; do \
		rm -rf certspotter-sd ; \
		mkdir -p certspotter-sd ; \
		cp -f $$file certspotter-sd/certspotter-sd ; \
		tar -czf $$file.tar.gz certspotter-sd ; \
		mv -f $$file.tar.gz ./ ; \
	done
	$Q cd $(OUT); sha256sum *.tar.gz > sha256sums.txt

# Misc

.PHONY: changelog
changelog: ## print changelog
	$Q printf "<!--title:$(VERSION) / $(DATE)-->\n# changelog\n"
ifneq "$(PREV_VERSION)" ""
	$Q git log --pretty="format:[\`%h\`](https://$(MODULE)/commit/%h) %s" $(PREV_VERSION)..$(VERSION)
else
	$Q git log --pretty="format:[\`%h\`](https://$(MODULE)/commit/%h) %s" $(VERSION)
endif

.PHONY: clean
clean: ; $(info $(M) cleaning...)	@ ## clean up everything
	$Q rm -rf $(BIN) $(OUT)

.PHONY: help
help:
	$Q grep -hE '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "%-17s %s\n", $$1, $$2}'

.PHONY: version
version: ## print current version
	$Q echo $(VERSION)
