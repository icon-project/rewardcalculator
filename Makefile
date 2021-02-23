#-------------------------------------------------------------------------------
#
# 	Makefile for building target binaries.
#

# Configuration
BUILD_ROOT = $(abspath ./)
BIN_DIR = ./bin
DST_DIR = /usr/local/bin

UNAME = $(shell uname)
INSTALL = install

GOBUILD = go build
GOTEST = go test
GOTOOL = go tool
GOMOD = go mod
GOCLEAN = go clean
GOBUILD_TAGS =
GOBUILD_ENVS = CGO_ENABLED=0 GO111MODULE=on
GOBUILD_LDFLAGS = -ldflags ""
GOBUILD_FLAGS = -mod vendor $(GOBUILD_TAGS) $(GOBUILD_LDFLAGS)

# Build flags
GL_VERSION ?= $(shell git describe --always --tags --dirty)
GL_TAG ?= latest
BUILD_INFO = tags($(GOBUILD_TAGS))-$(shell date '+%Y-%m-%d-%H:%M:%S')

#
# Build scripts for command binaries.
#
CMDS_DIR = ./cmd
CMDS = $(patsubst $(CMDS_DIR)/%/., %, $(wildcard $(CMDS_DIR)/*/.))

# Build flags for each command
icon_rc_LDFLAGS = -ldflags "-X 'main.version=$(GL_VERSION)' -X 'main.build=$(BUILD_INFO)'"

BUILD_TARGETS = icon_rc rctool

.DEFAULT_GOAL := all
all : clean $(BUILD_TARGETS) ## Build the tools for current OS

.PHONY: $(CMDS) linux darwin help
$(CMDS) :
	$(if $($@_LDFLAGS), \
		$(eval GOBUILD_LDFLAGS=$($@_LDFLAGS)), \
		$(eval GOBUILD_LDFLAGS=) \
	)
	@ echo "[#] go build $(CMDS_DIR)/$@ as $(BIN_DIR)/$@"
	$(GOBUILD_ENVS) $(GOBUILD) $(GOBUILD_FLAGS) -o $(BIN_DIR)/$@ $(CMDS_DIR)/$@

linux : ## Build the tools for linux
	@ echo "[#] build for $@"
	@ make GOBUILD_ENVS="$(GOBUILD_ENVS) GOOS=$@ GOARCH=amd64"

darwin : ## Build the tools for OS X
	@ echo "[#] build for $@"
	@ make GOBUILD_ENVS="$(GOBUILD_ENVS) GOOS=$@ GOARCH=amd64"

test : install ## Run unittest
	$(GOTEST) -test.short ./...

test_cov : ## Run unittest with code coverage
	$(GOTEST) -coverprofile cp.out ./...

test_cov_view : ## View unittest code coverage result
	$(GOTOOL) cover -html=./cp.out

modules : ## Update modules in vendor/
	$(GOMOD) tidy
	$(GOMOD) vendor

install : ## Install the tools on system
	@ \
	for target in $(BUILD_TARGETS); do \
		echo "[#] install $(BIN_DIR)/$$target to $(DST_DIR)"; \
		$(INSTALL) -m 755 $(BIN_DIR)/$$target $(DST_DIR); \
	done

clean : ## Remove generated files
	$(RM) -r $(BIN_DIR)
	$(GOCLEAN) -testcache

TARGET_MAX_CHAR_NUM=20
help : ## This help message
	@echo ''
	@echo 'Usage:'
	@echo '  make <target>'
	@echo ''
	@echo 'Targets:'
	@IFS=$$'\n' ; \
	help_lines=(`fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//'`); \
	for help_line in $${help_lines[@]}; do \
	IFS=$$'#' ; \
		help_split=($$help_line) ; \
		help_command=`echo $${help_split[0]} | sed -e 's/:.*//' -e 's/^ *//' -e 's/ *$$//'` ; \
		help_info=`echo $${help_split[2]} | sed -e 's/^ *//' -e 's/ *$$//'` ; \
		printf "  %-20s %s %s\n" $$help_command $$help_info; \
	done
