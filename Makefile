appname := trovilo

sources = $(shell find . -type f -name '*.go' -not -path "./vendor/*")
artifact_version = $(shell cat VERSION | tr -d '\n')
build_version = $(artifact_version)-$(shell date +%Y%m%d-%H%M%S)+$(shell git rev-parse --short HEAD)

build = cd cmd/trovilo/ && GOOS=$(1) GOARCH=$(2) go build -ldflags "-X=main.build=$(build_version)" -o ../../build/$(appname)$(3)
tar = cd build && tar -cvzf $(appname)-$(artifact_version).$(1)-$(2).tar.gz $(appname)$(3) && rm $(appname)$(3)
zip = cd build && zip $(appname)-$(artifact_version).$(1)-$(2).zip $(appname)$(3) && rm $(appname)$(3)

.PHONY: all test clean fmt vendor windows darwin linux

all: vendor test windows darwin linux

#test:
#	./test/run-integration-tests.sh

test:
	go test -v ./...

clean:
	rm -rf build/

fmt:
	@gofmt -l -w $(sources)

vendor:
	dep ensure

##### LINUX #####
linux: build/$(appname)-$(artifact_version).linux-amd64.tar.gz

build/$(appname)-$(artifact_version).linux-amd64.tar.gz: $(sources)
	$(call build,linux,amd64,)
	$(call tar,linux,amd64)

##### DARWIN (MAC) #####
darwin: build/$(appname)-$(artifact_version).darwin-amd64.tar.gz

build/$(appname)-$(artifact_version).darwin-amd64.tar.gz: $(sources)
	$(call build,darwin,amd64,)
	$(call tar,darwin,amd64)

##### WINDOWS #####
windows: build/$(appname)-$(artifact_version).windows-amd64.zip

build/$(appname)-$(artifact_version).windows-amd64.zip: $(sources)
	$(call build,windows,amd64,.exe)
	$(call zip,windows,amd64,.exe)
