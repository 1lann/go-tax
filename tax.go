package main

import (
	"encoding/json"
	"fmt"
	"github.com/1lann/go-tax/pdf"
	"io/ioutil"
	"os"
	"strings"
)

var holders []string

func display(filename string) {
	result, err := pdf.Process(filename, holders)
	if err != nil {
		fmt.Println("error processing:", err)
	}

	res, _ := json.MarshalIndent(result, "", "    ")
	fmt.Println(string(res))
}

func main() {
	file, err := os.Open("holders.txt")
	if err != nil {
		panic(err)
	}

	data, err := ioutil.ReadAll(file)
	file.Close()
	if err != nil {
		panic(err)
	}

	holders = strings.Split(string(data), "\n")
	if holders[len(holders)-1] == "" {
		holders = holders[:len(holders)-1]
	}

	display("samples/sample.pdf")
	display("samples/sample1.pdf")
	display("samples/sample2.pdf")
	display("samples/sample3.pdf")
}
