
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*")
UNITTEST_PACKAGES = $(shell go list ./... | grep -v /vendor/ | grep -v integration_test)
IMG_REPO ?= mintel/event-exporter
IMG_TAG ?= latest
BIN ?= event-exporter

all: fmt vet build

fmt:
	gofmt -l -w ${GOFILES_NOVENDOR}

dep:
	@which dep 2>/dev/null || $(GO) get github.com/golang/dep/cmd/dep
	dep ensure

build: dep
	go build -ldflags -s -v -o bin/${BIN} .

run: build
	bin/${BIN}

test:
	go test -ldflags -s -v --cover ${UNITTEST_PACKAGES}
	go vet ${UNITTEST_PACKAGES}

image:
	docker build -t ${IMG_REPO}:${IMG_TAG} .

push:
	docker push ${IMG_REPO}:${IMG_TAG}

docker: image push
