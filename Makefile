.PHONY: build install clean tidy pack

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
DIST    := dist

build:
	go build -o fbmcp ./cmd/fbmcp
	GOOS=windows GOARCH=amd64 go build -o fbmcp.exe cmd/fbmcp/main.go

install:
	go install ./cmd/fbmcp

tidy:
	go mod tidy

# Empacota binários de Linux e Windows em dist/ para anexar a um release no GitHub
pack:
	rm -rf $(DIST)
	mkdir -p $(DIST)/linux-amd64 $(DIST)/windows-amd64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o $(DIST)/linux-amd64/fbmcp ./cmd/fbmcp
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o $(DIST)/windows-amd64/fbmcp.exe ./cmd/fbmcp
	cp README.md $(DIST)/linux-amd64/
	cp README.md $(DIST)/windows-amd64/
	tar -czf $(DIST)/fbmcp-$(VERSION)-linux-amd64.tar.gz -C $(DIST)/linux-amd64 fbmcp README.md
	cd $(DIST)/windows-amd64 && zip -q ../fbmcp-$(VERSION)-windows-amd64.zip fbmcp.exe README.md
	rm -rf $(DIST)/linux-amd64 $(DIST)/windows-amd64
	cd $(DIST) && sha256sum fbmcp-$(VERSION)-* > checksums.txt
	@ls -lh $(DIST)

clean:
	rm -rf fbmcp fbmcp.exe $(DIST)
