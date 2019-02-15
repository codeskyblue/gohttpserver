package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	dkignore "github.com/codeskyblue/dockerignore"
	"golang.org/x/text/encoding/simplifiedchinese"
)

type Zip struct {
	*zip.Writer
}

func sanitizedName(filename string) string {
	if len(filename) > 1 && filename[1] == ':' &&
		runtime.GOOS == "windows" {
		filename = filename[2:]
	}
	filename = strings.TrimLeft(strings.Replace(filename, `\`, "/", -1), `/`)
	filename = filepath.ToSlash(filename)
	filename = filepath.Clean(filename)
	return filename
}

func statFile(filename string) (info os.FileInfo, reader io.ReadCloser, err error) {
	info, err = os.Lstat(filename)
	if err != nil {
		return
	}
	// content
	if info.Mode()&os.ModeSymlink != 0 {
		var target string
		target, err = os.Readlink(filename)
		if err != nil {
			return
		}
		reader = ioutil.NopCloser(bytes.NewBuffer([]byte(target)))
	} else if !info.IsDir() {
		reader, err = os.Open(filename)
		if err != nil {
			return
		}
	} else {
		reader = ioutil.NopCloser(bytes.NewBuffer(nil))
	}
	return
}

func (z *Zip) Add(relpath, abspath string) error {
	info, rdc, err := statFile(abspath)
	if err != nil {
		return err
	}
	defer rdc.Close()

	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	hdr.Name = sanitizedName(relpath)
	if info.IsDir() {
		hdr.Name += "/"
	}
	hdr.Method = zip.Deflate // compress method
	writer, err := z.CreateHeader(hdr)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, rdc)
	return err
}

func CompressToZip(w http.ResponseWriter, rootDir string) {
	rootDir = filepath.Clean(rootDir)
	zipFileName := filepath.Base(rootDir) + ".zip"

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+zipFileName+`"`)

	zw := &Zip{Writer: zip.NewWriter(w)}
	defer zw.Close()

	filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		zipPath := path[len(rootDir):]
		if info.Name() == YAMLCONF { // ignore .ghs.yml for security
			return nil
		}
		return zw.Add(zipPath, path)
	})
}

func ExtractFromZip(zipFile, path string, w io.Writer) (err error) {
	cf, err := zip.OpenReader(zipFile)
	if err != nil {
		return
	}
	defer cf.Close()

	rd := ioutil.NopCloser(bytes.NewBufferString(path))
	patterns, err := dkignore.ReadIgnore(rd)
	if err != nil {
		return
	}

	for _, file := range cf.File {
		matched, _ := dkignore.Matches(file.Name, patterns)
		if !matched {
			continue
		}
		rc, er := file.Open()
		if er != nil {
			err = er
			return
		}
		defer rc.Close()
		_, err = io.Copy(w, rc)
		if err != nil {
			return
		}
		return
	}
	return fmt.Errorf("File %s not found", strconv.Quote(path))
}

func unzipFile(filename, dest string) error {
	zr, err := zip.OpenReader(filename)
	if err != nil {
		return err
	}
	defer zr.Close()

	if dest == "" {
		dest = filepath.Dir(filename)
	}

	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		// ignore .ghs.yml
		filename := sanitizedName(f.Name)
		if filepath.Base(filename) == ".ghs.yml" {
			continue
		}
		fpath := filepath.Join(dest, filename)

		// filename maybe GBK or UTF-8
		// Ref: https://studygolang.com/articles/3114
		if f.Flags&(1<<11) == 0 { // GBK
			tr := simplifiedchinese.GB18030.NewDecoder()
			fpathUtf8, err := tr.String(fpath)
			if err == nil {
				fpath = fpathUtf8
			}
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		os.MkdirAll(filepath.Dir(fpath), os.ModePerm)
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		_, err = io.Copy(outFile, rc)
		outFile.Close()

		if err != nil {
			return err
		}
	}
	return nil
}
