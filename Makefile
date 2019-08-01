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
GOBUILD_TAGS = -tags ""
GOBUILD_ENVS = CGO_ENABLED=0 GO111MODULE=on
GOBUILD_LDFLAGS = -ldflags ""
GOBUILD_FLAGS = -mod vendor $(GOBUILD_TAGS) $(GOBUILD_LDFLAGS)

ifeq ($(UNAME),Linux)
	GOBUILD_ENVS += GOOS=linux GOARCH=amd64
endif

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
all : clean $(BUILD_TARGETS)

.PHONY: $(CMDS)
$(CMDS) :
	$(if $($@_LDFLAGS), \
		$(eval GOBUILD_LDFLAGS=$($@_LDFLAGS)), \
		$(eval GOBUILD_LDFLAGS=) \
	)
	echo "[#] go build $(CMDS_DIR)/$@"
	$(GOBUILD_ENVS) $(GOBUILD) $(GOBUILD_FLAGS) -o $(BIN_DIR)/$@ $(CMDS_DIR)/$@

test :
	$(GOTEST) -test.short ./...

test_cov :
	$(GOTEST) -coverprofile cp.out ./...

test_cov_view :
	$(GOTOOL) cover -html=./cp.out

modules :
	$(GOMOD) tidy
	$(GOMOD) vendor

install :
	@ \
	for target in $(BUILD_TARGETS); do \
		echo "[#] install $$target to $(DST_DIR)"; \
		$(INSTALL) -m 755 $(BIN_DIR)/$$target $(DST_DIR); \
	done

clean :
	@$(RM) -r $(BIN_DIR)
