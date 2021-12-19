package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mackerelio/mackerel-client-go"
)

type Capability struct {
	OS             string `json:"os"`
	OSVersion      string `json:"os_version"`
	Browser        string `json:"browser"`
	BrowserVersion string `json:"browser_version"`
	Device         string `json:"device"`
	Resolution     string `json:"resolution"`
}

type Scenario struct {
	Action       string     `json:"action"`
	ID           int64      `json:"id"`
	StartedAt    string     `json:"started_at"`
	FinishedAt   string     `json:"finished_at"`
	Status       string     `json:"status"`
	URL          string     `json:"url"`
	ScenarioID   int64      `json:"scenario_id"`
	ScenarioName string     `json:"scenario_name"`
	ReviewNeeded bool       `json:"review_needed"`
	TestPlanID   int64      `json:"test_plan_id"`
	Capability   Capability `json:"capability"`
}

type TestPlan struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type TestPlanWebhookFromAutify struct {
	Action       string     `json:"action"`
	ID           int64      `json:"id"`
	TestPlan     TestPlan   `json:"test_plan"`
	StartedAt    string     `json:"started_at"`
	FinishedAt   string     `json:"finished_at"`
	Status       string     `json:"status"`
	ReviewNeeded bool       `json:"review_needed"`
	URL          string     `json:"url"`
	Scenarios    []Scenario `json:"scenarios"`
}

func main() {
	log.Print("starting server...")
	http.HandleFunc("/autify2mackerel", handler)

	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}

	// Start HTTP server.
	log.Printf("listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	var testplanWebhookFromAutify TestPlanWebhookFromAutify
	byteArray, _ := ioutil.ReadAll(r.Body)
	body := string(byteArray)
	err := json.Unmarshal(byteArray, &testplanWebhookFromAutify)
	if err != nil {
		log.Fatalln(fmt.Sprintf("something wrong. detail: %s", body))
		w.WriteHeader(http.StatusBadRequest)
		return
	} else if testplanWebhookFromAutify.Scenarios == nil {
		log.Fatalln(fmt.Sprintf("Received TestScenario webhook. detail: %s", body))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// calculate each status test count
	var notPassedTestCount int
	var passedTestCount int
	for _, v := range testplanWebhookFromAutify.Scenarios {
		if v.Status == "passed" {
			passedTestCount++
		} else {
			notPassedTestCount++
		}
	}

	apikey := os.Getenv("MACKEREL_APIKEY")
	if apikey == "" {
		log.Fatalln("Mackerel API Key is required.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	serviceName := os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		serviceName = "hoge"
	}

	// post as mackerel service metrics
	client := mackerel.NewClient(apikey)
	nowUnixTime := time.Now().Unix()

	var metricsValues []*mackerel.MetricValue
	notPassedMetricValue := &mackerel.MetricValue{
		Name:  fmt.Sprintf("%s.autify.tests.not_passed", serviceName),
		Time:  nowUnixTime,
		Value: notPassedTestCount,
	}
	passedMetricValue := &mackerel.MetricValue{
		Name:  fmt.Sprintf("%s.autify.tests.passed", serviceName),
		Time:  nowUnixTime,
		Value: passedTestCount,
	}
	metricsValues = append(metricsValues, notPassedMetricValue, passedMetricValue)
	client.PostServiceMetricValues(serviceName, metricsValues)

	w.WriteHeader(http.StatusOK)
	return
}
