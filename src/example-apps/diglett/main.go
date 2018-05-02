package main

import (
	"example-apps/diglett/handlers"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"time"
)

var (
	queryTimeRegexp       *regexp.Regexp
	queryTimePrefixLength int
	answerSectionRegexp   *regexp.Regexp
)

func launchHandler(port int, statsHandler http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/stats", statsHandler)
	mux.Handle("/", &handlers.InfoHandler{
		Port: port,
	})

	go func() {
		http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), mux)
	}()
}

func getQueryTime(digOutput []byte) string {
	queryTimeLine := queryTimeRegexp.Find(digOutput)
	if len(queryTimeLine) < queryTimePrefixLength {
		return "INVALID TIME"
	}
	return string(queryTimeLine[queryTimePrefixLength:])
}

func getNumAnswers(digOutput []byte) int {
	return len(answerSectionRegexp.FindAll(digOutput, 500))
}
func main() {
	var err error
	queryTimeRegexp, err = regexp.Compile("Query time: \\d+.*")
	if err != nil {
		panic(err) // not tested
	}
	queryTimePrefixLength = len("Query time: ")

	systemPortString := os.Getenv("PORT")
	systemPort, err := strconv.Atoi(systemPortString)
	if err != nil {
		log.Fatal("invalid required env var PORT")
	}

	destination := os.Getenv("DIGLETT_DESTINATION")
	if destination == "" {
		log.Fatal("invalid required env var DIGLETT_DESTINATION")
	}
	answerSectionRegexp, err = regexp.Compile(destination + `.*IN.*A`)
	if err != nil {
		panic(err) // not tested
	}

	frequencyString := os.Getenv("DIGLETT_FREQUENCY_MS")
	frequency, err := strconv.Atoi(frequencyString)
	if err != nil {
		log.Fatal("invalid required env var DIGLETT_FREQUENCY_MS")
	}

	stats := &handlers.Stats{
		Latency: []float64{},
	}
	statsHandler := &handlers.StatsHandler{
		Stats: stats,
	}

	err = os.Setenv("PATH", "/bin:/usr/bin")
	if err != nil {
		log.Fatal("unable to set PATH") // not tested
	}

	launchHandler(systemPort, statsHandler)

	log.SetOutput(os.Stdout)
	ticker := time.NewTicker(time.Duration(frequency) * time.Millisecond)
	for range ticker.C {
		cmd := exec.Command("dig", destination, "+noall", "+answer", "+stats")
		output, err := cmd.Output()
		if err != nil {
			panic(err) // not tested
		}

		log.Printf("dig %s %s %d answers\n", destination, getQueryTime(output), getNumAnswers(output))
	}
}
