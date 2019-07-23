# NOTE: Adding a target so it shows up in the help listing
#    - The description is the text that is echoed in the first command in the target.
#    - Only 'public' targets (start with an alphanumeric character) display in the help listing.
#    - All public targets need a description

export CGO = 0

export GO111MODULE := on

.PHONY: help
help:
	@echo "describe make commands"
	@echo ""
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null |\
	  awk -v RS= -F: \
	    '/^# File/,/^# Finished Make data base/ {if ($$1 ~ /^[a-zA-Z]/) {printf "%-20s%s\n", $$1, substr($$9, 9, length($$9)-9)}}' |\
	  sort

.PHONY: test
test:
	@echo "run tests"
	go test ./cmd/... ./commands/... ./do/... ./install/... ./pkg/... ./pluginhost/... .

.PHONY: clean
clean:
	@echo "remove build / release artifacts"
	@rm -rf builds

.PHONY: vendor
vendor:
	@echo "vendor dependencies"
	go mod vendor
	go mod tidy

.PHONY: changelog
changelog:
	@echo "generate changelog entries"
	scripts/changelog.sh

.PHONY: mocks
mocks:
	@echo "update mocks"
	scripts/regenmocks.sh

.PHONY: shellcheck
shellcheck:
	@echo "analyze shell scripts"
	scripts/shell_check.sh

.PHONY: version
version:
	@echo "doctl version"
	scripts/version.sh

.PHONY: install-sembump
install-sembump:
	@echo "install/update sembump tool"
	@GO111MODULE=off go get -u github.com/jessfraz/junk/sembump

.PHONY: bump-and-tag
bump-and-tag: install-sembump
	@echo "BUMP=<patch|feature|breaking> bump and tag version"
	scripts/bumpversion.sh

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
	@echo "build local version"
	@mv $(OUT_D)/doctl_$(GOOS)_$(GOARCH) $(OUT_D)/doctl
	@echo "built $(OUT_D)/doctl"

.PHONY: _build_linux_amd64
_build_linux_amd64: GOOS = linux
_build_linux_amd64: GOARCH = amd64
_build_linux_amd64: _build

.PHONY: _base_docker_cntr
_base_docker_cntr:
	docker build -f Dockerfile.build . -t doctl_builder

.PHONY: docker_build
docker_build: _base_docker_cntr
	@echo "build doctl in local docker container"
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
