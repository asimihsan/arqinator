EXTERNAL_DEPENDENCIES := \
	github.com/aws/aws-sdk-go/... \
	google.golang.org/cloud/... \
	github.com/mattn/go-plist \
	github.com/mitchellh/go-homedir \
	golang.org/x/crypto/pbkdf2 \
	github.com/codegangsta/cli \
	github.com/Sirupsen/logrus \
	github.com/dustin/go-humanize \
	github.com/pkg/sftp

INTERNAL_DEPENDENCIES := \
	github.com/asimihsan/arqinator/arq \
	github.com/asimihsan/arqinator/arq/types \
	github.com/asimihsan/arqinator/crypto \
	github.com/asimihsan/arqinator/connector \

all: external-deps build

all-linux: deps build-linux

build: internal-deps
	go install github.com/asimihsan/arqinator

release: build-mac-386 build-mac-amd64 build-linux-386 build-linux-amd64 build-windows-386 build-windows-amd64

build-mac-386: internal-deps
	GOOS=darwin GOARCH=386 go build -o build/mac/386/arqinator
	cd build/mac/386 && tar cvf - arqinator | pigz -9 --force > arqinator.tar.gz && cd -

build-mac-amd64: internal-deps
	GOOS=darwin GOARCH=amd64 go build -o build/mac/amd64/arqinator
	cd build/mac/amd64 && tar cvf - arqinator | pigz -9 --force > arqinator.tar.gz && cd -

build-linux-386: internal-deps
	GOOS=linux GOARCH=386 go build -o build/linux/386/arqinator
	cd build/linux/386 && tar cvf - arqinator | pigz -9 --force > arqinator.tar.gz && cd -

build-linux-amd64: internal-deps
	GOOS=linux GOARCH=amd64 go build -o build/linux/amd64/arqinator
	cd build/linux/amd64 && tar cvf - arqinator | pigz -9 --force > arqinator.tar.gz && cd -

build-windows-386: internal-deps
	GOOS=windows GOARCH=386 go build -o build/windows/386/arqinator.exe
	cd build/windows/386 && tar cvf - arqinator.exe | pigz -9 --force > arqinator.tar.gz && cd -

build-windows-amd64: internal-deps
	GOOS=windows GOARCH=amd64 go build -o build/windows/amd64/arqinator.exe
	cd build/windows/amd64 && tar cvf - arqinator.exe | pigz -9 --force > arqinator.tar.gz && cd -

internal-deps:
	go build $(INTERNAL_DEPENDENCIES)

external-deps:
	go get $(EXTERNAL_DEPENDENCIES)
