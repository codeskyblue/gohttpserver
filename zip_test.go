package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractFromZip(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	err := ExtractFromZip("testdata/test.zip", "**/foo.txt", buf)
	assert.Nil(t, err)
	t.Log("Content: " + buf.String())
}

//func TestUnzipTo(t *testing.T){
//	err := unzipFile("testdata.zip", "./tmp")
//	assert.Nil(t, err)
//}
