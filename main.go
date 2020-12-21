package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strconv"
)

func sliceContains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

type P1Frame struct {
	powerConsumedT1  float64
	powerConsumedT2  float64
	powerDeliveredT1 float64
	powerDeliveredT2 float64
	currentTariff    float64
	currentInput     float64
	currentOutput    float64
	gasConsumed      float64
}

type P1DataReader interface {
	Read() string
}

type P1TxtReader struct {
	filePath string
}

func (p *P1TxtReader) Read() string {
	content, err := ioutil.ReadFile(p.filePath)
	if err != nil {
		log.Fatal(err)
	}
	return string(content)
}

func p1DataToFrame(data string) P1Frame {
	frame := P1Frame{}
	toSkip := []string{"0-0:96.1.1", "0-1:96.1.0", "0-1:24.1.0"}
	pat := regexp.MustCompile(`([0-1]\-\d:\d+.\d+.\d+)\((.+?)(\*.*?)?\)`)
	matches := pat.FindAllStringSubmatch(data, -1)
	for _, match := range matches {
		if sliceContains(toSkip, match[1]) {
			continue
		}
		f, err := strconv.ParseFloat(match[2], 64)
		if err != nil {
			log.Fatalf("Could not parse (%s).\n Error: %s", match[0], err)
		}
		switch key := match[1]; key {
		case "1-0:1.8.1":
			frame.powerConsumedT1 = f
		case "1-0:1.8.2":
			frame.powerConsumedT2 = f
		case "1-0:2.8.1":
			frame.powerDeliveredT1 = f
		case "1-0:2.8.2":
			frame.powerDeliveredT2 = f
		case "0-0:96.14.0":
			frame.currentTariff = f
		case "1-0:1.7.0":
			frame.currentInput = f
		case "1-0:2.7.0":
			frame.currentOutput = f
		case "0-1:24.3.0":
			frame.gasConsumed = f
		default:
			log.Fatalf("Could not find key (%s) for value (%s).\n", match[1], match[2])
		}
	}
	return frame
}
func main() {

	p := P1TxtReader{filePath: "example.txt"}
	data := p.Read()
	fmt.Println(data)

	frame := p1DataToFrame(data)

	fmt.Printf("%+v", frame)

}
