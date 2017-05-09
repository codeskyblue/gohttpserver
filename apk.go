package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"

	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/pkg/errors"
	"github.com/shogo82148/androidbinary"
)

type ApkInfo struct {
	PackageName  string `json:"packageName"`
	MainActivity string `json:"mainActivity"`
}

type Apk struct {
	filename string
	manifest Manifest
	table    *androidbinary.TableFile
}

func NewApk(filename string) (*Apk, error) {
	apk := &Apk{
		filename: filename,
	}
	if err := apk.parseManifest(); err != nil {
		return nil, errors.Wrap(err, "parse-manifest")
	}
	if err := apk.parseResources(); err != nil {
		return nil, err
	}
	apk.loadResourceNames()
	// log.Println(apk.Icon())
	return apk, nil
}

func (k *Apk) Icon() (image.Image, error) {
	imgData, err := k.readZipFile(k.manifest.App.Icon)
	if err != nil {
		return nil, err
	}

	m, _, err := image.Decode(bytes.NewReader(imgData))
	return m, err
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
			if intent.Action.Name == "android.intent.action.MAIN" {
				return act.Name, nil
			}
		}
	}
	return "", errors.New("No main activity found")
}

func (k *Apk) loadResourceNames() {
	k.manifest.App.Icon = k.getResource(k.manifest.App.Icon)
	k.manifest.App.Label = k.getResource(k.manifest.App.Label)
	for index, activity := range k.manifest.App.Activity {
		k.manifest.App.Activity[index].Label = k.getResource(activity.Label)
	}
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

func (k *Apk) getResource(id string) string {
	var resId uint32
	_, err := fmt.Sscanf(id, "@0x%x", &resId)
	if err != nil {
		return id
	}
	config := &androidbinary.ResTableConfig{}
	val, err := k.table.GetResource(androidbinary.ResId(resId), config)
	if err != nil {
		return id
	}
	return fmt.Sprintf("%s", val)
}

func (k *Apk) readZipFile(name string) (data []byte, err error) {
	buf := bytes.NewBuffer(nil)
	cf, err := zip.OpenReader(k.filename)
	if err != nil {
		return
	}
	defer cf.Close()
	for _, file := range cf.File {
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
