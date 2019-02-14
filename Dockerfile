FROM golang:1.11.5-alpine3.9

RUN apk add --no-cache make git && \
	go get -u github.com/golang/dep/cmd/dep && \
	mkdir -p /go/src/github.com/mintel/event-exporter

COPY Gopkg.lock Gopkg.toml /go/src/github.com/mintel/event-exporter/

WORKDIR /go/src/github.com/mintel/event-exporter/
RUN dep ensure -vendor-only

COPY . /go/src/github.com/mintel/event-exporter/

RUN go build -o /event-exporter

FROM alpine:3.9
COPY --from=0 /event-exporter /
CMD ["/event-exporter", "-v", "4"]
