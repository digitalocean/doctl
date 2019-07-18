# need to set go version to at least 1.11

export CGO = 0

export GO111MODULE := on

list:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null |\
	  awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' |\
	  sort |\
	  egrep -v -e '^[^[:alnum:]]' -e '^$@$$'

.PHONY: test
test:
	go test ./cmd/... ./commands/... ./do/... ./install/... ./pkg/... ./pluginhost/... .

.PHONY: clean
clean:
	@rm -rf builds

.PHONY: vendor
vendor:
	go mod vendor
	go mod tidy

.PHONY: changelog
changelog:
	scripts/changelog.sh

.PHONY: mocks
mocks:
	scripts/regenmocks.sh

# These builds are for convenience. This logic isn't used in the build-release process
my_d = $(shell pwd)
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
