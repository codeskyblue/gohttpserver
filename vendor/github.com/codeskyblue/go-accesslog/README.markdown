Custom format HTTP access logger in golang
==========================================

## Description

A library to build your own HTTP access logger.

## Usage

Provide a class that implements `accesslog.Logger` interface to make a logging HTTP handler.

``` golang
type LogRecord struct {
	Time                                time.Time
	Ip, Method, Uri, Protocol, Username string
	Status                              int
	Size                                int64
	ElapsedTime                         time.Duration
	CustomRecords                       map[string]string
}

type Logger interface {
	Log(record LogRecord)
}
```

## Example

``` golang
import (
	"log"
	"net/http"

	accesslog "github.com/mash/go-accesslog"
)

type logger struct {
}

func (l logger) Log(record accesslog.LogRecord) {
	log.Println(record.Method + " " + record.Uri)
}

func main() {
	l := logger{}
	handler := http.FileServer(http.Dir("."))
	http.ListenAndServe(":8080", accesslog.NewLoggingHandler(handler, l))
}
```
