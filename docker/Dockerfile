FROM golang:1.15
WORKDIR /app/gohttpserver
ADD . /app/gohttpserver
ENV GOPROXY="https://goproxy.cn,https://goproxy.io"
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-X main.VERSION=docker' -o gohttpserver

FROM library/alpine:3.11.6
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add --update --no-cache ca-certificates
WORKDIR /app
RUN mkdir -p /app/public
VOLUME /app/public
ADD assets ./assets
COPY --from=0 /app/gohttpserver/gohttpserver .
EXPOSE 8000
ENTRYPOINT [ "/app/gohttpserver", "--root=/app/public" ]
CMD []
