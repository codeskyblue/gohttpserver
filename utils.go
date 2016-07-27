package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func formatSize(file os.FileInfo) string {
	if file.IsDir() {
		return "-"
	}
	size := file.Size()
	switch {
	case size > 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(size)/1024/1024)
	case size > 1024:
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	default:
		return strconv.Itoa(int(size)) + " B"
	}
	return ""
}

func getRealIP(req *http.Request) string {
	xip := req.Header.Get("X-Real-IP")
	if xip == "" {
		xip = strings.Split(req.RemoteAddr, ":")[0]
	}
	return xip
}
