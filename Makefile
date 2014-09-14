LAST_TAG := $(shell git describe --abbrev=0 --tags)
PREV_VERSION := $(shell git tag -l | egrep '\d+.\d+\.\d+' | tail -2 | head -1)

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

UPLOAD_CMD = $(GHRELEASE) upload -u $(USER) -r $(EXECUTABLE) -t $(LAST_TAG) -n $(subst /,-,$(FILE)) -f bin/$(FILE)

all: test $(EXECUTABLE)

# binaries
bin/freebsd/amd64/$(EXECUTABLE): update_internal_version .deps
	GOARCH=amd64 GOOS=freebsd go build -o "$@"
bin/darwin/amd64/$(EXECUTABLE): update_internal_version .deps
	GOARCH=amd64 GOOS=darwin go build -o "$@"
bin/linux/amd64/$(EXECUTABLE): update_internal_version .deps
	GOARCH=amd64 GOOS=linux go build -o "$@"
bin/windows/amd64/$(EXECUTABLE).exe: update_internal_version .deps
	GOARCH=amd64 GOOS=windows go build -o "$@"

%.tar.bz2: %
	tar -C $(shell dirname $@) -jcvf "$<.tar.bz2" $(shell basename $<)
%.zip: %.exe
	zip -j "$@" "$<"

release: test $(COMPRESSED_EXECUTABLE_TARGETS) $(GOPATH)/bin/github-release releaselog-$(LAST_TAG).txt 
	git push && git push --tags
	$(GHRELEASE) release -u $(USER) -r $(EXECUTABLE) \
		-t $(LAST_TAG) -n $(LAST_TAG) -d "`cat releaselog-$(LAST_TAG).txt`" || true
	$(foreach FILE,$(COMPRESSED_EXECUTABLES),$(UPLOAD_CMD);)

releaselog-$(LAST_TAG).txt:
	git shortlog $(PREV_VERSION)..$(LAST_TAG) > releaselog-$(LAST_TAG).txt

.deps: $(GOPATH)/bin/godep
	godep restore
	touch .deps

update_internal_version: doctl.go
	sed -i '' 's/const AppVersion = ".*"/const AppVersion = "$(LAST_TAG)"/' doctl.go

$(EXECUTABLE): .deps
	go build -o "$@"

install: .deps
	go install

$(GOPATH)/bin/godep:
	go get github.com/tools/godep

$(GOPATH)/bin/github-release:
	go get github.com/aktau/github-release

clean:
	rm .deps
	rm -rf bin/

test: .deps
	go test -v ./...

.PHONY: clean release install test update_internal_version
