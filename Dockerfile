FROM golang:1.9 AS build
WORKDIR /go/src/github.com/codeskyblue/gohttpserver
ADD . /go/src/github.com/codeskyblue/gohttpserver/
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gohttpserver .

FROM alpine:3.6
WORKDIR /app
RUN mkdir -p /app/public
VOLUME /app/public
ADD res ./res
COPY --from=build /go/src/github.com/codeskyblue/gohttpserver/gohttpserver .
EXPOSE 8000
CMD ["/app/gohttpserver", "--root=/app/public"]
