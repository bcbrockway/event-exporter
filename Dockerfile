FROM golang:1.11.5-alpine3.9

ENV SRC_PATH $GOPATH/src/github.com/mintel/event-exporter
RUN apk add --no-cache make git
ADD . $SRC_PATH/
RUN echo $SRC_PATH && cd $SRC_PATH && make build


FROM alpine:3.9
COPY --from=0 /go/src/github.com/mintel/event-exporter/bin/event-exporter /
CMD ["/event-exporter", "-v", "4"]
