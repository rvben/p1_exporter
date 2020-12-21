package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func sliceContains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

var (
	powerConsumed = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "power_consumed",
		Help: "The total power consumed in kWh",
	},
		[]string{"tariff"},
	)
	powerDelivered = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "power_delivered",
		Help: "The total power delivered in kWh",
	},
		[]string{"tariff"},
	)
	currentTariff = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "current_tariff",
		Help: "The power tariff currently in effect",
	})
	currentImport = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "current_import",
		Help: "The power currently being imported in kW",
	})
	currentExport = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "current_export",
		Help: "The power currently being exported in kW",
	})
	gasConsumed = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gas_consumed",
		Help: "The total gas consumed in m3",
	})
)

type P1Frame struct {
	powerConsumedT1  float64
	powerConsumedT2  float64
	powerDeliveredT1 float64
	powerDeliveredT2 float64
	currentTariff    float64
	currentImport    float64
	currentExport    float64
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
			powerConsumed.WithLabelValues("1").Set(f)
		case "1-0:1.8.2":
			frame.powerConsumedT2 = f
			powerConsumed.WithLabelValues("2").Set(f)
		case "1-0:2.8.1":
			frame.powerDeliveredT1 = f
			powerDelivered.WithLabelValues("1").Set(f)
		case "1-0:2.8.2":
			frame.powerDeliveredT2 = f
			powerDelivered.WithLabelValues("2").Set(f)
		case "0-0:96.14.0":
			frame.currentTariff = f
			currentTariff.Set(f)
		case "1-0:1.7.0":
			frame.currentImport = f
			currentImport.Set(f)
		case "1-0:2.7.0":
			frame.currentExport = f
			currentExport.Set(f)
		case "0-1:24.3.0":
			frame.gasConsumed = f
			gasConsumed.Set(f)
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

	registry := prometheus.NewRegistry()
	registerer := prometheus.WrapRegistererWithPrefix("p1_", registry)
	registerer.MustRegister(powerConsumed)
	registerer.MustRegister(powerDelivered)
	registerer.MustRegister(currentTariff)
	registerer.MustRegister(currentImport)
	registerer.MustRegister(currentExport)
	registerer.MustRegister(gasConsumed)

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	http.Handle("/metrics", handler)
	http.ListenAndServe(":2112", nil)
}
