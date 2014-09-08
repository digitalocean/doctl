VERSION := 0.0.3
LAST_TAG := $(shell git describe --abbrev=0 --tags)
PREV_VERSION := $(shell git tag -l | egrep '\d+.\d+\.\d+' | tail -2 | head -1)

GIT_LOG := $(shell git shortlog $(PREV_VERSION)..$(LAST_TAG))

USER := slantview
EXECUTABLE := doctl

GHRELEASE := github-release

UNIX_EXECUTABLES := \
	darwin/amd64/$(EXECUTABLE) \
	freebsd/amd64/$(EXECUTABLE) \
	linux/amd64/$(EXECUTABLE)
WIN_EXECUTABLES := \
	windows/amd64/$(EXECUTABLE).exe

COMPRESSED_EXECUTABLES=$(UNIX_EXECUTABLES:%=%.tar.bz2) $(WIN_EXECUTABLES:%.exe=%.zip)
COMPRESSED_EXECUTABLE_TARGETS=$(COMPRESSED_EXECUTABLES:%=bin/%)

UPLOAD_CMD = $(GHRELEASE) upload -u $(USER) -r $(EXECUTABLE) -t "$(GIT_LOG)" -n $(subst /,-,$(FILE)) -f bin/$(FILE)

all: $(EXECUTABLE)

# arm
bin/linux/arm/5/$(EXECUTABLE):
	GOARM=5 GOARCH=arm GOOS=linux go build -o "$@"
bin/linux/arm/7/$(EXECUTABLE):
	GOARM=7 GOARCH=arm GOOS=linux go build -o "$@"

# 386
bin/darwin/386/$(EXECUTABLE):
	GOARCH=386 GOOS=darwin go build -o "$@"
bin/linux/386/$(EXECUTABLE):
	GOARCH=386 GOOS=linux go build -o "$@"
bin/windows/386/$(EXECUTABLE):
	GOARCH=386 GOOS=windows go build -o "$@"

# amd64
bin/freebsd/amd64/$(EXECUTABLE):
	GOARCH=amd64 GOOS=freebsd go build -o "$@"
bin/darwin/amd64/$(EXECUTABLE):
	GOARCH=amd64 GOOS=darwin go build -o "$@"
bin/linux/amd64/$(EXECUTABLE):
	GOARCH=amd64 GOOS=linux go build -o "$@"
bin/windows/amd64/$(EXECUTABLE).exe:
	GOARCH=amd64 GOOS=windows go build -o "$@"

%.tar.bz2: %
	tar -jcvf "$<.tar.bz2" "$<"
%.zip: %.exe
	zip "$@" "$<"

release: $(COMPRESSED_EXECUTABLE_TARGETS) install_github_release test
	git push && git push --tags
	$(GHRELEASE) release -u $(USER) -r $(EXECUTABLE) \
		-t $(LAST_TAG) -n $(LAST_TAG) || true
	$(foreach FILE,$(COMPRESSED_EXECUTABLES),$(UPLOAD_CMD);)

.deps: install_godep
	godep restore
	touch .deps

$(EXECUTABLE): .deps
	go build -o "$@"

install:
	go install

install_godep:
	go get github.com/tools/godep

install_github_release:
	go get github.com/aktau/github-release

clean:
	rm .deps
	rm -rf bin/

test:
	go test -v ./...

.PHONY: clean release install test install_godep install_github_release
