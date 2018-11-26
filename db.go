package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

func panicErr(e error) {
	if e != nil {
		panic(e)
	}
}

type Model struct {
	Filename string
	Data     *map[string]int
	LastSave time.Time
}

var DBModel Model = Model{}
var SaveInterval float64 = 60.0 // seconds

func (model *Model) DBRead(filename string) *map[string]int {
	data := make(map[string]int)
	model.Filename = filename
	model.Data = &data

	// create if file not exist
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		f, cerr := os.Create(filename)
		panicErr(cerr)

		s, jerr := json.Marshal(data)
		panicErr(jerr)
		f.Write(s)
		defer f.Close()
		return &data
	}

	dat, err := ioutil.ReadFile(filename)
	panicErr(err)

	err = json.Unmarshal(dat, &data)
	panicErr(err)

	fmt.Println(data)
	return &data
}

func (model *Model) DBWrite() error {
	fmt.Println("db writing")
	dat, err := json.Marshal(model.Data)
	if err != nil {
		return err
	}
	tmp := model.Filename + ".un~"
	err = ioutil.WriteFile(tmp, dat, 0644)
	if err != nil {
		return err
	}
	return os.Rename(tmp, model.Filename)
}

func (model *Model) Incre(key string) {
	(*model.Data)[key]++
	fmt.Println(model.Data)
	go model.AutoSave()
}

func (model *Model) AutoSave() bool {
	if time.Since(model.LastSave).Seconds() > SaveInterval {
		model.DBWrite()
		model.LastSave = time.Now()
		return true
	}
	return false
}

type dldEntry struct {
	ip   string
	path string
}

// DldCounter is used to validate duplicated download requests
type DldCounter struct {
	IPHistory map[dldEntry]time.Time
}

var DCounter DldCounter = DldCounter{}
var DldInterval float64 = 60.0 // seconds

func (counter *DldCounter) Validate(ip string, path string) bool {
	if counter.IPHistory == nil {
		counter.IPHistory = make(map[dldEntry]time.Time)
	}
	entry := dldEntry{ip, path}
	if time.Since(counter.IPHistory[entry]).Seconds() > DldInterval {
		counter.IPHistory[entry] = time.Now()
		// fmt.Println("valid", ip)
		return true
	}
	// fmt.Println("invalid", ip)
	return false
}

// func main() {
// 	filename := "./mydb.json"
// 	DBModel.DBRead(filename)
// 	fmt.Println(DBModel.Filename)
// 	fmt.Println(*DBModel.Data)
// 	defer DBModel.DBWrite()
// 	(*DBModel.Data)["b"] = 2
// 	(*DBModel.Data)["a"] += 2
// 	fmt.Println(DBModel.Data)
// }
