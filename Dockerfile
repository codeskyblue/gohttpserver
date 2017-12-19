FROM golang:latest
RUN mkdir /app 
WORKDIR /app 
ENV SRC_DIR=/go/src/github.com/codeskyblue/gohttpserver
ADD . $SRC_DIR
RUN cd $SRC_DIR; go build; cp gohttpserver /app/
ENTRYPOINT ["/app/gohttpserver"]
