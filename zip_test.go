package main

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExtractFromZip(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	err := ExtractFromZip("testdata/test.zip", "**/foo.txt", buf)
	assert.NotNil(t, err)
	t.Log("Content: " + buf.String())
}

//func TestUnzipTo(t *testing.T){
//	err := unzipFile("testdata.zip", "./tmp")
//	assert.Nil(t, err)
//}