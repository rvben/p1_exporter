package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tarm/serial"
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

type P1SerialReader struct {
	portName string
	baud     int
}

func (p *P1SerialReader) Read() string {
	startR := regexp.MustCompile(`^\/`)
	endR := regexp.MustCompile(`^!`)
	var sb strings.Builder

	c := &serial.Config{Name: p.portName, Baud: p.baud}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(s)
	for scanner.Scan() {
		line := scanner.Text()
		if startR.MatchString(line) {
			sb.WriteString(line)
			break
		}
	}
	for scanner.Scan() {
		line := scanner.Text()
		sb.WriteString(line)
		if endR.MatchString(line) {
			break
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func setMetric(match []string) {
	toSkip := []string{"0-0:96.1.1", "0-1:96.1.0", "0-1:24.1.0", "1-3:0.2.8", "0-0:1.0.0", "1-0:99.97.0", "0-0:96.13.0"}
	if match != nil && !sliceContains(toSkip, match[1]) {
		f, err := strconv.ParseFloat(match[2], 64)
		if err != nil {
			log.Printf("Could not parse (%s).\n Error: %s\n", match[0], err)
		}
		switch key := match[1]; key {
		case "1-0:1.8.1":
			powerConsumed.WithLabelValues("1").Set(f)
		case "1-0:1.8.2":
			powerConsumed.WithLabelValues("2").Set(f)
		case "1-0:2.8.1":
			powerDelivered.WithLabelValues("1").Set(f)
		case "1-0:2.8.2":
			powerDelivered.WithLabelValues("2").Set(f)
		case "0-0:96.14.0":
			currentTariff.Set(f)
		case "1-0:1.7.0":
			currentImport.Set(f)
		case "1-0:2.7.0":
			currentExport.Set(f)
		case "0-1:24.3.0":
			gasConsumed.Set(f)
		case "0-1:24.2.1":
			gasConsumed.Set(f)
		default:
			// log.Printf("Could not find key (%s) for value (%s).\n", match[1], match[2])
		}
	}
}

func recordMetrics(scanner *bufio.Scanner) {
	pat := regexp.MustCompile(`([0-1]\-\d:\d+.\d+.\d+).*\((.+?)(\*.*?)?\)`)
	for scanner.Scan() {
		line := scanner.Text()
		match := pat.FindStringSubmatch(line)
		go setMetric(match)
	}
}

func main() {

	registry := prometheus.NewRegistry()
	registerer := prometheus.WrapRegistererWithPrefix("p1_", registry)
	registerer.MustRegister(powerConsumed)
	registerer.MustRegister(powerDelivered)
	registerer.MustRegister(currentTariff)
	registerer.MustRegister(currentImport)
	registerer.MustRegister(currentExport)
	registerer.MustRegister(gasConsumed)

	c := &serial.Config{Name: "/dev/ttyUSB0", Baud: 115200}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(s)
	go recordMetrics(scanner)

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	http.Handle("/metrics", handler)
	err = http.ListenAndServe(":2112", nil)
	if err != nil {
		log.Fatalf("\nError: %v", err)
	}
}
