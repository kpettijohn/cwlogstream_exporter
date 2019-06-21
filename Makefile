REVISION := $(shell git rev-list -1 HEAD)
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
VERSION := 0.0.1
BUILD_VARS := -X main.exporterBranch=$(BRANCH) -X main.exporterVersion=$(VERSION) -X main.exporterRevision=$(REVISION)

all: clean build

clean:
	rm -f ./bin/*

build: bin/cwlogstream_exporter-darwin-amd64 bin/cwlogstream_exporter-linux-amd64 bin/cwlogstream_exporter-windows-amd64.exe

bin/cwlogstream_exporter-darwin-amd64:
	GO111MODULE=on GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w $(BUILD_VARS)" -o ./bin/cwlogstream_exporter-darwin-amd64
bin/cwlogstream_exporter-linux-amd64:
	GO111MODULE=on GOOS=linux GOARCH=amd64 go build -ldflags="-s -w $(BUILD_VARS)" -o ./bin/cwlogstream_exporter-linux-amd64
bin/cwlogstream_exporter-windows-amd64.exe:
	GO111MODULE=on GOOS=windows GOARCH=amd64 go build -ldflags="-s -w $(BUILD_VARS)" -o ./bin/cwlogstream_exporter-windows-amd64.exe
