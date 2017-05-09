androidbinary
=====

[![Build Status](https://travis-ci.org/shogo82148/androidbinary.svg?branch=master)](https://travis-ci.org/shogo82148/androidbinary)

Android binary file parser

## Parse XML binary

``` go
package main

import (
	"fmt"
	"github.com/shogo82148/androidbinary"
	"os"
)

func main() {
	f, _ := os.Open("AndroidManifest")
	xml, _ := androidbinary.NewXMLFile(f)
	reader := xml.Reader()
	// read XML from reader
}
```

## License

This software is released under the MIT License, see LICENSE.
