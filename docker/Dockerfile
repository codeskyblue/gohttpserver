FROM golang:1.10
WORKDIR /go/src/github.com/codeskyblue/gohttpserver
ADD . /go/src/github.com/codeskyblue/gohttpserver/
RUN go get -v
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-X main.VERSION=docker' -o gohttpserver

FROM debian:stretch
WORKDIR /app
RUN mkdir -p /app/public
RUN apt-get update && apt-get install -y ca-certificates
VOLUME /app/public
ADD assets ./assets
COPY --from=0 /go/src/github.com/codeskyblue/gohttpserver/gohttpserver .
EXPOSE 8000
ENTRYPOINT [ "/app/gohttpserver", "--root=/app/public" ]
CMD []
