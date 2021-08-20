package main

import (
	"embed"
	"net/http"
)

//go:embed assets
var assetsFS embed.FS

// Assets contains project assets.
var Assets = http.FS(assetsFS)
