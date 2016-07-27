package plist

import "io"

type countedWriter struct {
	io.Writer
	nbytes int
}

func (w *countedWriter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	w.nbytes += n
	return n, err
}

func (w *countedWriter) BytesWritten() int {
	return w.nbytes
}
