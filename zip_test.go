package main

import (
	"bytes"
	"testing"
)

func TestExtractFromZip(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	err := ExtractFromZip("testdata/test.zip", "**/foo.txt", buf)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Content: " + buf.String())
}
