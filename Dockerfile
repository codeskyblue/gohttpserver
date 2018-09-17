FROM golang:1.10
WORKDIR /go/src/github.com/codeskyblue/gohttpserver
ADD . /go/src/github.com/codeskyblue/gohttpserver/
RUN go get -v
RUN CGO_ENABLED=0 GOOS=linux go build -o gohttpserver .

FROM debian:stretch
WORKDIR /app
RUN mkdir -p /app/public
VOLUME /app/public
ADD res ./res
ADD assets ./assets
COPY --from=0 /go/src/github.com/codeskyblue/gohttpserver/gohttpserver .
EXPOSE 8000
CMD ["/app/gohttpserver", "--root=/app/public"]
