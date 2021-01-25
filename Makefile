# NOTE: Adding a target so it shows up in the help listing
#    - The description is the text that is echoed in the first command in the target.
#    - Only 'public' targets (start with an alphanumeric character) display in the help listing.
#    - All public targets need a description

export CGO_ENABLED = 0

export GO111MODULE := on

# ORIGIN is used when testing release code
ORIGIN ?= origin
BUMP ?= patch

.PHONY: help
help:
	@echo "==> describe make commands"
	@echo ""
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null |\
	  awk -v RS= -F: \
	    '/^# File/,/^# Finished Make data base/ {if ($$1 ~ /^[a-zA-Z]/) {printf "%-20s%s\n", $$1, substr($$9, 9, length($$9)-9)}}' |\
	  sort

my_d = $(shell pwd)
OUT_D = $(shell echo $${OUT_D:-$(my_d)/builds})
DOCS_OUT = $(shell echo $${DOCS_OUT:-$(my_d)/builds/docs/yaml})

UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

GOOS = linux
ifeq ($(UNAME_S),Darwin)
  GOOS = darwin
endif

ifeq ($(GOARCH),)
  GOARCH = amd64
  ifneq ($(UNAME_M), x86_64)
    GOARCH = 386
  endif
endif

.PHONY: _build
_build:
	@echo "=> building doctl via go build"
	@echo ""
	@OUT_D=${OUT_D} GOOS=${GOOS} GOARCH=${GOARCH} scripts/_build.sh
	@echo "built $(OUT_D)/doctl_$(GOOS)_$(GOARCH)"

.PHONY: build
build: _build
	@echo "==> build local version"
	@echo ""
	@mv $(OUT_D)/doctl_$(GOOS)_$(GOARCH) $(OUT_D)/doctl
	@echo "installed as $(OUT_D)/doctl"

.PHONY: native
native: build
	@echo ""
	@echo "==> The 'native' target is deprecated. Use 'make build'"

.PHONY: _build_linux_amd64
_build_linux_amd64: GOOS = linux
_build_linux_amd64: GOARCH = amd64
_build_linux_amd64: _build

.PHONY: docker_build
docker_build:
	@echo "==> build doctl in local docker container"
	@echo ""
	@mkdir -p $(OUT_D)
	@docker build -f Dockerfile \
		--build-arg GOARCH=$(GOARCH) \
		. -t doctl_local
	@docker run --rm \
		-v $(OUT_D):/copy \
		-it --entrypoint /bin/cp \
		doctl_local /app/doctl /copy/
	@docker run --rm \
		-v $(OUT_D):/copy \
		-it --entrypoint /bin/chown \
		alpine -R $(shell whoami | id -u): /copy
	@echo "Built binaries to $(OUT_D)"
	@echo "Created a local Docker container. To use, run: docker run --rm -it doctl_local"

.PHONY: test_unit
test_unit:
	@echo "==> run unit tests"
	@echo ""
	go test -mod=vendor ./commands/... ./do/... ./pkg/... .

.PHONY: test_integration
test_integration:
	@echo "==> run integration tests"
	@echo ""
	go test -v -mod=vendor ./integration

.PHONY: test
test: test_unit test_integration

.PHONY: shellcheck
shellcheck:
	@echo "==> analyze shell scripts"
	@echo ""
	@scripts/shell_check.sh

.PHONY: gofmt_check
gofmt_check:
	@echo "==> ensure code adheres to gofmt (with vendor directory excluded)"
	@echo ""
	@GOFMT=$$(gofmt -l ${GOFILES_NOVENDOR}); \
	if [ -n "$${GOFMT}" ]; then \
		echo "gofmt checking failed:\n"; echo "$${GOFMT} \n"; exit 1; \
	fi

.PHONY: snap_image
snap_image:
	@echo "==> build docker image for releasing snap"
	@echo ""
	@scripts/snap_image.sh

.PHONY: _snap_image_version
_snap_image_version:
	@ORIGIN=${ORIGIN} SNAP_IMAGE=true scripts/version.sh

.PHONY: build_local_snap
build_local_snap:
	@echo "==> build local snap using local image tagged doctl-snap-base"
	@echo ""
	@BUILD=local_snap scripts/snap_image.sh

.PHONY: prerelease_snap_image
prerelease_snap_image:
	@echo "==> tag doctl-snap-base as a prerelease and push to dockerhub as latest"
	@echo ""
	@BUILD=pre scripts/snap_image.sh

.PHONY: finalize_snap_image
finalize_snap_image:
	@echo "==> tag latest with most recent doctl version push to dockerhub"
	@echo ""
	@ORIGIN=${ORIGIN} BUILD=finalize scripts/snap_image.sh

CHANNEL ?= stable

.PHONY: _build_snap
_build_snap:
	@CHANNEL=${CHANNEL} scripts/_build_snap.sh

.PHONY: snap
snap:
	@echo "==> publish snap (normally done by travis)"
	@echo ""
	@CHANNEL=${CHANNEL} scripts/snap.sh

.PHONY: mocks
mocks:
	@echo "==> update mocks"
	@echo ""
	@scripts/regenmocks.sh

.PHONY: _upgrade_godo
_upgrade_godo:
	go get -u github.com/digitalocean/godo

.PHONY: upgrade_godo
upgrade_godo: _upgrade_godo vendor mocks
	@echo "==> upgrade the godo version"
	@echo ""

.PHONY: vendor
vendor:
	@echo "==> vendor dependencies"
	@echo ""
	go mod vendor
	go mod tidy

.PHONY: clean
clean:
	@echo "==> remove build / release artifacts"
	@echo ""
	@rm -rf builds dist out parts prime stage doctl_v*.snap

.PHONY: _install_github_release_notes
_install_github_release_notes:
	@GO111MODULE=off go get -u github.com/digitalocean/github-changelog-generator

.PHONY: _changelog
_changelog: _install_github_release_notes
	@scripts/changelog.sh

.PHONY: changes
changes: _install_github_release_notes
	@echo "==> list merged PRs since last release"
	@echo ""
	@changes=$(shell scripts/changelog.sh) && cat $$changes && rm -f $$changes

.PHONY: version
version:
	@echo "==> doctl version"
	@echo ""
	@ORIGIN=${ORIGIN} scripts/version.sh

.PHONY: _install_sembump
_install_sembump:
	@echo "=> installing/updating sembump tool"
	@echo ""
	@GO111MODULE=off go get -u github.com/jessfraz/junk/sembump

.PHONY: tag
tag: _install_sembump
	@echo "==> BUMP=${BUMP} tag"
	@echo ""
	@ORIGIN=${ORIGIN} scripts/bumpversion.sh

.PHONY: _release
_release:
	@echo "=> releasing"
	@echo ""
	@scripts/release.sh

.PHONY: _tag_and_release
_tag_and_release: tag
	@echo "=> DEPRECATED: BUMP=${BUMP} tag and release"
	@echo ""
	@$(MAKE) _release
	@$(MAKE) snap

.PHONY: release
release:
	@echo "==> release (most recent tag, normally done by travis)"
	@echo ""
	@$(MAKE) _release

.PHONY: docs
docs:
	@echo "==> Generate YAML documentation in ${DOCS_OUT}"
	@echo ""
	@mkdir -p ${DOCS_OUT}
	@DOCS_OUT=${DOCS_OUT} go run scripts/gen-yaml-docs.go
