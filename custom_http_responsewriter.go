package main

import (
	"net/http"
)

// ResponseWriterDelegate is an http.ResponseWriter that keeps track
// of additional metadata.
type ResponseWriterDelegate struct {
	http.ResponseWriter
	header  http.Header
	Code    int
	Context string
}

func (d *ResponseWriterDelegate) Header() http.Header {
	//d.ResponseWriter.WriteHeader(code)
	if d.header == nil {
		d.header = make(http.Header)
	}
	return d.header
}

// WriteHeader and keep track of the http response code
func (d *ResponseWriterDelegate) WriteHeader(code int) {
	//d.ResponseWriter.WriteHeader(code)
	d.Code = code
}

// Write such that the first invocation will trigger an implicit WriteHeader(http.StatusOK)
func (d *ResponseWriterDelegate) Write(p []byte) (int, error) {
	if d.Code == 0 {
		d.Code = http.StatusOK
	}
	d.Context = d.Context + string(p)
	return len(d.Context), nil
}
