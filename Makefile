
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*")
UNITTEST_PACKAGES = $(shell go list ./... | grep -v /vendor/ | grep -v integration_test)


all: fmt vet build

fmt:
	gofmt -l -w ${GOFILES_NOVENDOR}

vet:
	go vet ${UNITTEST_PACKAGES}

build:
	go build -ldflags -s -v -o bin/event-exporter .

run: build
	bin/event-exporter

test:
	go test -ldflags -s -v --cover ${UNITTEST_PACKAGES}

image:
	docker build -t nabadger/event-exporter .

push:
	docker push nabadger/event-exporter

docker: image push
