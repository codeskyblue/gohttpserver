package httpexpect

import (
	"bytes"
	"fmt"
	"github.com/valyala/fasthttp"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
)

// Binder implements networkless http.RoundTripper attached directly to
// http.Handler.
//
// Binder emulates network communication by invoking given http.Handler
// directly. It passes httptest.ResponseRecorder as http.ResponseWriter
// to the handler, and then constructs http.Response from recorded data.
type Binder struct {
	handler http.Handler
}

// NewBinder returns a new Binder given a http.Handler.
//
// Example:
//   client := &http.Client{
//       Transport: NewBinder(handler),
//   }
func NewBinder(handler http.Handler) Binder {
	return Binder{handler}
}

// RoundTrip implements http.RoundTripper.RoundTrip.
func (binder Binder) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Proto == "" {
		req.Proto = fmt.Sprintf("HTTP/%d.%d", req.ProtoMajor, req.ProtoMinor)
	}

	if req.Body != nil {
		if req.ContentLength == -1 {
			req.TransferEncoding = []string{"chunked"}
		}
	} else {
		req.Body = ioutil.NopCloser(bytes.NewReader(nil))
	}

	recorder := httptest.NewRecorder()

	binder.handler.ServeHTTP(recorder, req)

	resp := http.Response{
		Request:    req,
		StatusCode: recorder.Code,
		Status:     http.StatusText(recorder.Code),
		Header:     recorder.HeaderMap,
	}

	if recorder.Flushed {
		resp.TransferEncoding = []string{"chunked"}
	}

	if recorder.Body != nil {
		resp.Body = ioutil.NopCloser(recorder.Body)
	}

	return &resp, nil
}

// FastBinder implements networkless http.RoundTripper attached directly
// to fasthttp.RequestHandler.
//
// FastBinder emulates network communication by invoking given http.Handler
// directly. It passes converts http.Request to fasthttp.Request, invokes
// handler, and then converts fasthttp.Response to http.Response.
type FastBinder struct {
	handler fasthttp.RequestHandler
}

// NewFastBinder returns a new FastBinder given a fasthttp.RequestHandler.
//
// Example:
//   client := &http.Client{
//       Transport: NewFastBinder(fasthandler),
//   }
func NewFastBinder(handler fasthttp.RequestHandler) FastBinder {
	return FastBinder{handler}
}

// RoundTrip implements http.RoundTripper.RoundTrip.
func (binder FastBinder) RoundTrip(stdreq *http.Request) (*http.Response, error) {
	var fastreq fasthttp.Request

	convertRequest(stdreq, &fastreq)

	var ctx fasthttp.RequestCtx

	ctx.Init(&fastreq, nil, nil)

	if stdreq.ContentLength >= 0 {
		ctx.Request.Header.SetContentLength(int(stdreq.ContentLength))
	} else {
		ctx.Request.Header.Add("Transfer-Encoding", "chunked")
	}

	if stdreq.Body != nil {
		b, err := ioutil.ReadAll(stdreq.Body)
		if err == nil {
			ctx.Request.SetBody(b)
		}
	}

	binder.handler(&ctx)

	return convertResponse(stdreq, &ctx.Response), nil
}

func convertRequest(stdreq *http.Request, fastreq *fasthttp.Request) {
	fastreq.SetRequestURI(stdreq.URL.String())

	fastreq.Header.SetMethod(stdreq.Method)

	for k, a := range stdreq.Header {
		for n, v := range a {
			if n == 0 {
				fastreq.Header.Set(k, v)
			} else {
				fastreq.Header.Add(k, v)
			}
		}
	}
}

func convertResponse(stdreq *http.Request, fastresp *fasthttp.Response) *http.Response {
	status := fastresp.Header.StatusCode()
	body := fastresp.Body()

	stdresp := &http.Response{
		Request:    stdreq,
		StatusCode: status,
		Status:     http.StatusText(status),
	}

	fastresp.Header.VisitAll(func(k, v []byte) {
		sk := string(k)
		sv := string(v)
		if stdresp.Header == nil {
			stdresp.Header = make(http.Header)
		}
		stdresp.Header.Add(sk, sv)
	})

	if fastresp.Header.ContentLength() == -1 {
		stdresp.TransferEncoding = []string{"chunked"}
	}

	if body != nil {
		stdresp.Body = ioutil.NopCloser(bytes.NewReader(body))
	} else {
		stdresp.Body = ioutil.NopCloser(bytes.NewReader(nil))
	}

	return stdresp
}
