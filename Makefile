#-------------------------------------------------------------------------------
#
# 	Makefile for building target binaries.
#

# Configuration
BUILD_ROOT = $(abspath ./)
BIN_DIR = ./bin
LINUX_BIN_DIR = ./linux
DST_DIR = /usr/local/bin

UNAME = $(shell uname)
INSTALL = install

GOBUILD = go build
GOTEST = go test
GOTOOL = go tool
GOMOD = go mod
GOBUILD_TAGS =
GOBUILD_ENVS = CGO_ENABLED=0 GO111MODULE=on
GOBUILD_LDFLAGS =
GOBUILD_FLAGS = -mod vendor -tags "$(GOBUILD_TAGS)" -ldflags "$(GOBUILD_LDFLAGS)"
GOBUILD_ENVS_LINUX = $(GOBUILD_ENVS) GOOS=linux GOARCH=amd64

# Build flags
GL_VERSION ?= $(shell git describe --always --tags --dirty)
GL_TAG ?= latest
BUILD_INFO = tags($(GOBUILD_TAGS))-$(shell date '+%Y-%m-%d-%H:%M:%S')

#
# Build scripts for command binaries.
#
CMDS = $(patsubst cmd/%,%,$(wildcard cmd/*))
.PHONY: $(CMDS)
define CMD_template
$(BIN_DIR)/$(1) : $(1)
$(1) : GOBUILD_LDFLAGS+=$$($(1)_LDFLAGS)
$(1) :
	@ \
	rm -f $(BIN_DIR)/$(1) ; \
	echo "[#] go build ./cmd/$(1)"
	$$(GOBUILD_ENVS) \
	$$(GOBUILD) $$(GOBUILD_FLAGS) \
	    -o $(BIN_DIR)/$(1) ./cmd/$(1)

$(LINUX_BIN_DIR)/$(1) : $(1)-linux
$(1)-linux : GOBUILD_LDFLAGS+=$$($(1)_LDFLAGS)
$(1)-linux :
	@ \
	rm -f $(LINUX_BIN_DIR)/$(1) ; \
	echo "[#] go build ./cmd/$(1)"
	$$(GOBUILD_ENVS_LINUX) \
	go build $$(GOBUILD_FLAGS) \
	    -o $(LINUX_BIN_DIR)/$(1) ./cmd/$(1)
endef
$(foreach M,$(CMDS),$(eval $(call CMD_template,$(M))))

# Build flags for each command
icon_rc_LDFLAGS = -X 'main.version=$(GL_VERSION)' -X 'main.build=$(BUILD_INFO)'
BUILD_TARGETS += icon_rc

linux : $(addsuffix -linux,$(BUILD_TARGETS))

test :
	$(GOTEST) -test.short ./...

test_cov :
	$(GOTEST) -coverprofile cp.out ./...

test_cov_view :
	$(GOTOOL) cover -html=./cp.out

modules :
	$(GOMOD) vendor

install :
ifeq ($(UNAME),Darwin)
	$(INSTALL) -m 644 $(BIN_DIR)/$(BUILD_TARGETS) $(DST_DIR)
else
	$(INSTALL) -m 644 $(LINUX_BIN_DIR)/$(BUILD_TARGETS) $(DST_DIR)
endif

.DEFAULT_GOAL := all
all : $(BUILD_TARGETS)
