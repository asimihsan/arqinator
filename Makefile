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
	GOOS=darwin GOARCH=386 go build -o build/mac32/arqinator
	pigz -9 --force --keep build/mac32/arqinator

build-mac-64: internal-deps
	GOOS=darwin GOARCH=amd64 go build -o build/mac64/arqinator
	pigz -9 --force --keep build/mac64/arqinator

build-linux-32: internal-deps
	GOOS=linux GOARCH=386 go build -o build/linux32/arqinator
	pigz -9 --force --keep build/linux32/arqinator

build-linux-64: internal-deps
	GOOS=linux GOARCH=amd64 go build -o build/linux64/arqinator
	pigz -9 --force --keep build/linux64/arqinator

build-windows-32: internal-deps
	GOOS=windows GOARCH=386 go build -o build/windows32/arqinator.exe
	pigz -9 --force --keep build/windows32/arqinator.exe
	mv build/windows32/arqinator.exe.gz build/windows32/arqinator.gz

build-windows-64: internal-deps
	GOOS=windows GOARCH=amd64 go build -o build/windows64/arqinator.exe
	pigz -9 --force --keep build/windows64/arqinator.exe
	mv build/windows64/arqinator.exe.gz build/windows64/arqinator.gz

internal-deps:
	go build $(INTERNAL_DEPENDENCIES)

external-deps:
	go get $(EXTERNAL_DEPENDENCIES)
