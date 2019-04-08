.PHONY: build
export CGO=0

my_d=$(shell pwd)
OUT_D = $(shell echo $${OUT_D:-$(my_d)/builds})

GOOS = linux
UNAME_S := $(shell uname -s)
UNAME_P := $(shell uname -p)
ifeq ($(UNAME_S),Darwin)
  GOOS = darwin
  GOARCH = 386
endif

GOARCH = amd64
ifneq ($(UNAME_P), x86_64)
  GOARCH = 386
endif

native: _build
native:
	@mv $(OUT_D)/doctl_$(GOOS)_$(GOARCH) $(OUT_D)/doctl
	@echo "built $(OUT_D)/doctl"

_build:
	@mkdir -p builds
	@echo "building doctl"
	@cd cmd/doctl && env GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(OUT_D)/doctl_$(GOOS)_$(GOARCH)
	@echo "built $(OUT_D)/doctl_$(GOOS)_$(GOARCH)"


build_mac: GOOS = darwin
build_mac: GOARCH = 386
build_mac: _build

build_linux_386: GOARCH = 386
build_linux_386: _build

build_linux_amd64: GOARCH = amd64
build_linux_amd64: _build

clean:
	@rm -rf builds

test:
	go test ./cmd/... ./commands/... ./do/... ./install/... ./pkg/... ./pluginhost/... .

_base_docker_cntr:
	docker build -f Dockerfile.build . -t doctl_builder

docker_build: _base_docker_cntr
docker_build:
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
