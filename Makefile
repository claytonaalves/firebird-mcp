.PHONY: build install clean tidy

build:
	go build -o fbmcp ./cmd/fbmcp

install:
	go install ./cmd/fbmcp

tidy:
	go mod tidy

clean:
	rm -f fbmcp
