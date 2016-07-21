// Package interpol provides utility functions for doing format-string like
// string interpolation using named parameters.
// Currently, a template only accepts variable placeholders delimited by brace
// characters (eg. "Hello {foo} {bar}").
package interpol

import (
	"bytes"
	"errors"
	"io"
)

// Errors returned when formatting templates.
var (
	ErrUnexpectedClose = errors.New("interpol: unexpected close in template")
	ErrExpectingClose  = errors.New("interpol: expecting close in template")
	ErrKeyNotFound     = errors.New("interpol: key not found")
)

// Func receives the placeholder key and writes to the io.Writer. If an error
// happens, the function can return an error, in which case the interpolation
// will be aborted.
type Func func(key string, w io.Writer) error

// WithFunc interpolates the specified template with replacements using the
// given function.
func WithFunc(template string, format Func) (string, error) {
	f := newInterpolator(template)
	f.format = format
	if err := f.interpolate(); err != nil {
		return "", err
	}
	return f.buffer.String(), nil
}

// WithMap interpolates the specified template with replacements using the
// given map. If a placeholder is used for which a value is not found, an error
// is returned.
func WithMap(template string, m map[string]string) (string, error) {
	format := func(key string, w io.Writer) error {
		value, ok := m[key]
		if !ok {
			return ErrKeyNotFound
		}
		_, err := w.Write([]byte(value))
		return err
	}
	return WithFunc(template, format)
}

type interpolator struct {
	template string
	buffer   *bytes.Buffer
	format   Func
	start    int
	closing  bool
}

func newInterpolator(template string) *interpolator {
	buffer := bytes.NewBuffer(nil)
	buffer.Grow(len(template))
	return &interpolator{
		template: template,
		buffer:   buffer,
		start:    -1,
		closing:  false,
	}
}

func (f *interpolator) open(i int) error {
	if f.closing {
		return ErrUnexpectedClose
	}
	if f.start >= 0 {
		f.buffer.WriteRune('{')
		f.start = -1
	} else {
		f.start = i + 1
	}
	return nil
}

func (f *interpolator) close(i int) error {
	if f.start >= 0 {
		if err := f.format(f.template[f.start:i], f.buffer); err != nil {
			return err
		}
		f.start = -1
	} else if f.closing {
		f.closing = false
	} else {
		f.closing = true
		f.buffer.WriteRune('}')
	}
	return nil
}

func (f *interpolator) interpolate() error {
	var err error
	for i, t := range f.template {
		switch t {
		case '{':
			err = f.open(i)
		case '}':
			err = f.close(i)
		default:
			err = f.append(t)
		}
		if err != nil {
			return err
		}
	}
	return f.finish()
}

func (f *interpolator) finish() error {
	if f.start >= 0 {
		return ErrExpectingClose
	}
	if f.closing {
		return ErrUnexpectedClose
	}
	return nil
}

func (f *interpolator) append(t rune) error {
	if f.closing {
		return ErrUnexpectedClose
	}
	if f.start < 0 {
		f.buffer.WriteRune(t)
	}
	return nil
}
