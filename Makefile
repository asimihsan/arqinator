EXTERNAL_DEPENDENCIES := \
	github.com/aws/aws-sdk-go/... \
	google.golang.org/cloud/... \
	github.com/edsrzf/mmap-go \
	github.com/mattn/go-plist \
	github.com/mitchellh/go-homedir \
	golang.org/x/crypto/pbkdf2 \
	github.com/codegangsta/cli \
	github.com/Sirupsen/logrus \
	github.com/dustin/go-humanize

INTERNAL_DEPENDENCIES := \
	github.com/asimihsan/arqinator/arq \
	github.com/asimihsan/arqinator/arq/types \
	github.com/asimihsan/arqinator/crypto \
	github.com/asimihsan/arqinator/connector \

all: external-deps build

all-linux: deps build-linux

build: internal-deps
	go install github.com/asimihsan/arqinator

release: build-mac-32 build-mac-64 build-linux-32 build-linux-64 build-windows-32 build-windows-64

build-mac-32: internal-deps
	GOOS=darwin GOARCH=386 go build -o build/arqinator-mac-32

build-mac-64: internal-deps
	GOOS=darwin GOARCH=amd64 go build -o build/arqinator-mac-64

build-linux-32: internal-deps
	GOOS=linux GOARCH=386 go build -o build/arqinator-linux-32

build-linux-64: internal-deps
	GOOS=linux GOARCH=amd64 go build -o build/arqinator-linux-64

build-windows-32: internal-deps
	GOOS=windows GOARCH=386 go build -o build/arqinator-windows-32.exe

build-windows-64: internal-deps
	GOOS=windows GOARCH=amd64 go build -o build/arqinator-windows-64.exe

internal-deps:
	go build $(INTERNAL_DEPENDENCIES)

external-deps:
	go get $(EXTERNAL_DEPENDENCIES)
