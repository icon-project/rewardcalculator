#-------------------------------------------------------------------------------
#
# 	Makefile for building target binaries.
#

# Configuration
BUILD_ROOT = $(abspath ./)
BIN_DIR = ./bin
LINUX_BIN_DIR = ./linux

GOBUILD = go build
GOTEST = go test
GOTOOL = go tool
GOBUILD_TAGS =
GOBUILD_ENVS = CGO_ENABLED=0
GOBUILD_LDFLAGS =
GOBUILD_FLAGS = -tags "$(GOBUILD_TAGS)" -ldflags "$(GOBUILD_LDFLAGS)"
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
$(1) : | vendor
	@ \
	rm -f $(BIN_DIR)/$(1) ; \
	echo "[#] go build ./cmd/$(1)"
	$$(GOBUILD_ENVS) \
	$$(GOBUILD) $$(GOBUILD_FLAGS) \
	    -o $(BIN_DIR)/$(1) ./cmd/$(1)

$(LINUX_BIN_DIR)/$(1) : $(1)-linux
$(1)-linux : GOBUILD_LDFLAGS+=$$($(1)_LDFLAGS)
$(1)-linux : | vendor
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

vendor :
	@ \
	$(MAKE) ensure
ensure :
	@ \
	echo "[#] dep ensure"
	dep ensure

linux : $(addsuffix -linux,$(BUILD_TARGETS))

# TODO support docker
#DOCKER_IMAGE_TAG ?= latest
#GOLOOP_ENV_IMAGE = goloop-env:$(GL_TAG)
#GOCHAIN_IMAGE = gochain:$(GL_TAG)
#GOCHAIN_DOCKER_DIR = $(BUILD_ROOT)/build/gochain/
#GOLOOP_BASE_PATH = /work/src/github.com/icon-project/goloop
#GOLOOP_GOPATH = /work
#
#goloop-env-image :
#	@ \
#	if [ "`docker images -q $(GOLOOP_ENV_IMAGE)`" == "" ] ; then \
#	    docker build -t $(GOLOOP_ENV_IMAGE) ./docker/goloop-env/ ; \
#	fi
#
#run-% : goloop-env-image
#	@ \
#	docker run -it --rm \
#	    -v $(BUILD_ROOT):$(GOLOOP_BASE_PATH) \
#	    -w $(GOLOOP_BASE_PATH) \
#	    -e "GOPATH=$(GOLOOP_GOPATH)" \
#	    $(GOLOOP_ENV_IMAGE) \
#	    make "GL_VERSION=$(GL_VERSION)" "BUILD_INFO=$(BUILD_INFO)" \
#		$(patsubst run-%,%,$@)
#
#gochain-image: run-gochain-linux
#	@ rm -rf $(GOCHAIN_DOCKER_DIR)
#	@ mkdir -p $(GOCHAIN_DOCKER_DIR)
#	@ cp $(BUILD_ROOT)/docker/gochain/* $(GOCHAIN_DOCKER_DIR)
#	@ cp $(BUILD_ROOT)/linux/gochain $(GOCHAIN_DOCKER_DIR)
#	@ docker build -t $(GOCHAIN_IMAGE) $(GOCHAIN_DOCKER_DIR)

test :
	$(GOTEST) -test.short ./...

test_cov :
	$(GOTEST) -coverprofile cp.out ./...

test_cov_view :
	$(GOTOOL) cover -html=./cp.out

.DEFAULT_GOAL := all
all : $(BUILD_TARGETS)
