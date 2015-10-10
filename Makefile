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

release: build-mac-386 build-mac-amd64 build-linux-386 build-linux-amd64 build-windows-386 build-windows-amd64 build-freebsd-386 build-freebsd-amd64 build-netbsd-386 build-netbsd-amd64 build-openbsd-386 build-openbsd-amd64 build-solaris-amd64

build-mac-386: internal-deps
	GOOS=darwin GOARCH=386 go build -o build/mac/386/arqinator
	pigz -9 --force --keep build/mac/386/arqinator

build-mac-amd64: internal-deps
	GOOS=darwin GOARCH=amd64 go build -o build/mac/amd64/arqinator
	pigz -9 --force --keep build/mac/amd64/arqinator

build-linux-386: internal-deps
	GOOS=linux GOARCH=386 go build -o build/linux/386/arqinator
	pigz -9 --force --keep build/linux/386/arqinator

build-linux-amd64: internal-deps
	GOOS=linux GOARCH=amd64 go build -o build/linux/amd64/arqinator
	pigz -9 --force --keep build/linux/amd64/arqinator

build-windows-386: internal-deps
	GOOS=windows GOARCH=386 go build -o build/windows/386/arqinator.exe
	pigz -9 --force --keep build/windows/386/arqinator.exe
	mv build/windows/386/arqinator.exe.gz build/windows/386/arqinator.gz

build-windows-amd64: internal-deps
	GOOS=windows GOARCH=amd64 go build -o build/windows/amd64/arqinator.exe
	pigz -9 --force --keep build/windows/amd64/arqinator.exe
	mv build/windows/amd64/arqinator.exe.gz build/windows/amd64/arqinator.gz

internal-deps:
	go build $(INTERNAL_DEPENDENCIES)

external-deps:
	go get $(EXTERNAL_DEPENDENCIES)
