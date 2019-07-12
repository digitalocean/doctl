# need to set go version to at least 1.11

export CGO=0

export GO111MODULE := on

# These builds are for convenience. This logic isn't used in the build-release process
my_d=$(shell pwd)
OUT_D = $(shell echo $${OUT_D:-$(my_d)/builds})

UNAME_S := $(shell uname -s)
UNAME_P := $(shell uname -p)

GOOS = linux
ifeq ($(UNAME_S),Darwin)
  GOOS = darwin
endif

GOARCH = amd64
ifneq ($(UNAME_P), x86_64)
  GOARCH = 386
endif

.PHONY: _build
_build:
	@mkdir -p builds
	@echo "building doctl via go build"
	@cd cmd/doctl && env GOOS=$(GOOS) GOARCH=$(GOARCH) GOFLAGS=-mod=vendor \
	  go build -o $(OUT_D)/doctl_$(GOOS)_$(GOARCH)
	@echo "built $(OUT_D)/doctl_$(GOOS)_$(GOARCH)"

.PHONY: native
native: _build
	@mv $(OUT_D)/doctl_$(GOOS)_$(GOARCH) $(OUT_D)/doctl
	@echo "built $(OUT_D)/doctl"

# end convenience builds

# docker targets for developing in docker
.PHONY: build_mac
build_mac: GOOS = darwin
build_mac: GOARCH = 386
build_mac: _build

.PHONY: build_linux_386
build_linux_386: GOOS = linux
build_linux_386: GOARCH = 386
build_linux_386: _build

.PHONY: build_linux_amd64
build_linux_amd64: GOOS = linux
build_linux_amd64: GOARCH = amd64
build_linux_amd64: _build

.PHONY: _base_docker_cntr
_base_docker_cntr:
	docker build -f Dockerfile.build . -t doctl_builder

.PHONY: docker_build
docker_build: _base_docker_cntr
	@mkdir -p $(OUT_D)
	@docker build -f Dockerfile.cntr . -t doctl_local
	@docker run --rm \
		-v $(OUT_D):/copy \
		-it --entrypoint /usr/bin/rsync \
		doctl_local -av /app/ /copy/
	@docker run --rm \
		-v $(OUT_D):/copy \
		-it --entrypoint /bin/chown \
		alpine -R $(shell whoami | id -u): /copy
	@echo "Built binaries to $(OUT_D)"
	@echo "Created a local Docker container. To use, run: docker run --rm -it doctl_local"
# end docker targets

.PHONY: clean
clean:
	@rm -rf builds

.PHONY: test
test:
	go test ./cmd/... ./commands/... ./do/... ./install/... ./pkg/... ./pluginhost/... .

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor
