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

build-linux: internal-deps
	GOOS=linux GOARCH=amd64 go install github.com/asimihsan/arqinator

internal-deps:
	go build $(INTERNAL_DEPENDENCIES)

external-deps:
	go get $(EXTERNAL_DEPENDENCIES)
