package apk

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	_ "image/jpeg"
	_ "image/png"

	"github.com/pkg/errors"
	"github.com/shogo82148/androidbinary"
)

var DefaultResTableConfig = &androidbinary.ResTableConfig{}

type Apk struct {
	f         *os.File
	zipreader *zip.Reader
	manifest  Manifest
	table     *androidbinary.TableFile
}

// OpenFile will open the file specified by filename and return Apk
func OpenFile(filename string) (apk *Apk, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			f.Close()
		}
	}()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	apk, err = OpenZipReader(f, fi.Size())
	if err != nil {
		return nil, err
	}
	apk.f = f
	return
}

// OpenZipReader has same arguments like zip.NewReader
func OpenZipReader(r io.ReaderAt, size int64) (*Apk, error) {
	zipreader, err := zip.NewReader(r, size)
	if err != nil {
		return nil, err
	}
	apk := &Apk{
		zipreader: zipreader,
	}
	if err = apk.parseManifest(); err != nil {
		return nil, errors.Wrap(err, "parse-manifest")
	}
	if err = apk.parseResources(); err != nil {
		return nil, err
	}
	return apk, nil
}

// Close is avaliable only if apk is created with OpenFile
func (k *Apk) Close() error {
	if k.f == nil {
		return nil
	}
	return k.f.Close()
}

// Icon return icon image
func (k *Apk) Icon(resConfig *androidbinary.ResTableConfig) (image.Image, error) {
	iconPath := k.getResource(k.manifest.App.Icon, resConfig)
	if strings.HasPrefix(iconPath, "@0x") {
		return nil, errors.New("unable to convert icon-id to icon path")
	}
	imgData, err := k.readZipFile(iconPath)
	if err != nil {
		return nil, err
	}
	m, _, err := image.Decode(bytes.NewReader(imgData))
	return m, err
}

func (k *Apk) Label(resConfig *androidbinary.ResTableConfig) (s string, err error) {
	s = k.getResource(k.manifest.App.Label, resConfig)
	if strings.HasPrefix(s, "@0x") {
		err = errors.New("unable to convert label-id to string")
	}
	return
}

func (k *Apk) Manifest() Manifest {
	return k.manifest
}

func (k *Apk) PackageName() string {
	return k.manifest.Package
}

func (k *Apk) MainAcitivty() (activity string, err error) {
	for _, act := range k.manifest.App.Activity {
		for _, intent := range act.IntentFilter {
			if intent.Action.Name == "android.intent.action.MAIN" &&
				intent.Category.Name == "android.intent.category.LAUNCHER" {
				return act.Name, nil
			}
		}
	}
	return "", errors.New("No main activity found")
}

func (k *Apk) parseManifest() error {
	xmlData, err := k.readZipFile("AndroidManifest.xml")
	if err != nil {
		return errors.Wrap(err, "read-manifest.xml")
	}
	xmlfile, err := androidbinary.NewXMLFile(bytes.NewReader(xmlData))
	if err != nil {
		return errors.Wrap(err, "parse-axml")
	}
	reader := xmlfile.Reader()
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	return xml.Unmarshal(data, &k.manifest)
}

func (k *Apk) parseResources() (err error) {
	resData, err := k.readZipFile("resources.arsc")
	if err != nil {
		return
	}
	k.table, err = androidbinary.NewTableFile(bytes.NewReader(resData))
	return
}

func (k *Apk) getResource(id string, resConfig *androidbinary.ResTableConfig) string {
	if resConfig == nil {
		resConfig = DefaultResTableConfig
	}
	var resId uint32
	_, err := fmt.Sscanf(id, "@0x%x", &resId)
	if err != nil {
		return id
	}
	val, err := k.table.GetResource(androidbinary.ResId(resId), resConfig)
	if err != nil {
		return id
	}
	return fmt.Sprintf("%s", val)
}

func (k *Apk) readZipFile(name string) (data []byte, err error) {
	buf := bytes.NewBuffer(nil)
	for _, file := range k.zipreader.File {
		if file.Name != name {
			continue
		}
		rc, er := file.Open()
		if er != nil {
			err = er
			return
		}
		defer rc.Close()
		_, err = io.Copy(buf, rc)
		if err != nil {
			return
		}
		return buf.Bytes(), nil
	}
	return nil, fmt.Errorf("File %s not found", strconv.Quote(name))
}
