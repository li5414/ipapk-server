BINARY := ipapk-server
VERSION ?= v1.0.0
PLATFORMS := windows darwin

.PHONY: $(PLATFORMS)
$(PLATFORMS):
	CGO_ENABLED=1 GO15VENDOREXPERIMENT=1 GOOS=$@ GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY)-$@

.PHONY: linux
linux:
	CC=x86_64-linux-musl-gcc CXX=x86_64-linux-musl-g++ CGO_ENABLED=1 GO15VENDOREXPERIMENT=1 GOOS=linux GOARCH=amd64 go build -v -ldflags  "-X main.version=$(VERSION)" -o $(BINARY)-linux

.PHONY: release
release: windows linux darwin