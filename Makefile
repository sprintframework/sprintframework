BIN := sprint
IMAGE := sprint
TARGET := target
VERSION := $(shell git describe --tags --always --dirty)
TAG := $(VERSION)
REGISTRY := codeallergy
PWD := $(shell pwd)
NOW := $(shell date +"%m-%d-%Y")

all: build

version:
	@echo $(TAG)

deps:
	go install github.com/codeallergy/go-bindata/go-bindata@v1.0.0

bindata:
	go-bindata -pkg resources -o pkg/resources/bindata.go -nocompress -nomemcopy -fs -prefix "resources/" resources/...
	go-bindata -pkg assetsgz -o pkg/assetsgz/bindata.go -nounpack -nomemcopy -fs -prefix "assets/" assets/...
	go-bindata -pkg assets -o pkg/assets/bindata.go -nocompress -nomemcopy -fs -prefix "assets/" assets/...

build: bindata
	rm -rf rsrc.syso
	go mod tidy
	go test -cover ./...
	go build -o $(BIN)_darwin -v -ldflags "-X main.Version=$(VERSION) -X main.Build=$(NOW)"

update:
	go get -u ./...

target: build
	rm -rf $(TARGET)
	mkdir $(TARGET)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(TARGET)/$(BIN)_linux -v -ldflags "-X main.Version=$(VERSION) -X main.Build=$(NOW)"
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o $(TARGET)/$(BIN)_darwin -v -ldflags "-X main.Version=$(VERSION) -X main.Build=$(NOW)"
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o $(TARGET)/$(BIN).exe -v -ldflags "-X main.Version=$(VERSION) -X main.Build=$(NOW)"

run: build
	env COS=dev ./$(BIN)

test: build
	env COS=test ./$(BIN)

docker:
	docker build --build-arg VERSION=$(VERSION) --build-arg BUILD=$(NOW) -t $(REGISTRY)/$(IMAGE):$(TAG) -f Dockerfile .

docker-run: docker
	docker run -it --rm -p 8080:8080 -p 8443:8443 -p 8444:8444 --env SAUCE_BOOT --env SAUCE_AUTH -v $(PWD)/db:/app/db -v $(PWD)/log:/app/log $(REGISTRY)/$(IMAGE):$(TAG) /app/bin/sprint run

docker-build:
	mkdir -p $(TARGET)
	rm -rf $(TARGET)/$(BIN)_linux
	docker build --build-arg VERSION=$(VERSION) --build-arg BUILD=$(NOW) -t $(REGISTRY)/$(IMAGE):$(TAG)-build -f Dockerfile.build .
	docker run --rm $(REGISTRY)/$(IMAGE):$(TAG)-build > $(TARGET)/$(BIN)_linux

docker-push: docker
	docker push ${REGISTRY}/${IMAGE}:${TAG}
	docker tag ${REGISTRY}/${IMAGE}:${TAG} ${REGISTRY}/${IMAGE}:latest
	docker push ${REGISTRY}/${IMAGE}:latest

docker-target:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(TARGET)/$(BIN)_linux -v -ldflags "-X main.Version=$(VERSION) -X main.Build=$(NOW)"
	docker run -it --rm -v /Users/ashvid/go/src/github.com/sprintframework/sprintframework/target:/target ubuntu:16.04

clean:
	docker ps -q -f 'status=exited' | xargs docker rm
	echo "y" | docker system prune

licenses:
	go-licenses csv "github.com/sprintframework/sprintframework" > resources/licenses.txt



