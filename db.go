package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

func panicErr(e error) {
	if e != nil {
		panic(e)
	}
}

type Model struct {
	Filename string
	Data     *map[string]int
}

var DBModel Model = Model{}

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

func (model *Model) DBWrite() {
	fmt.Println("writing")
	dat, err := json.Marshal(*model.Data)
	panicErr(err)
	err = ioutil.WriteFile(model.Filename, dat, 0644)
	panicErr(err)
}

func (model *Model) Incre(key string) {
	(*model.Data)[key]++
	fmt.Println(model.Data)
	go model.DBWrite()
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
