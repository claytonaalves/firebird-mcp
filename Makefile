.PHONY: build install clean tidy

build:
	go build -o fbmcp ./cmd/fbmcp
	GOOS=windows GOARCH=amd64 go build -o fbmcp.exe cmd/fbmcp/main.go

install:
	go install ./cmd/fbmcp

tidy:
	go mod tidy

clean:
	rm -f fbmcp
