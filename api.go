/*

	Mesostic API Front

	/app - API endpoint
	/ping - a readiness check
	/metrics - prometheus metrics

*/

package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

// define prometheus metrics
var msgPostCnt = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "mesostic_post_app_total",
	Help: "Total number of POST me-api requests.",
})

var pingCnt = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "mesostic_ping_total",
	Help: "Total number of Readiness pings.",
})

var msgPostDur = prometheus.NewHistogram(prometheus.HistogramOpts{
	Name: "mesostic_post_app_timer_seconds",
	Help: "Historgram for the runtime of POST to /app",
	// 50 Buckets, 10ms each, starting at 1ms
	Buckets: prometheus.LinearBuckets(0.001, 0.01, 50),
})

// data object model
type Submit struct {
	Text        string
	SpineString string
}

// FSubmit ::: POST Method for multi-part form submission.
func FSubmit(w http.ResponseWriter, r *http.Request) {
	msgPostCnt.Add(1)
	msgTimer := prometheus.NewTimer(msgPostDur)
	defer msgTimer.ObserveDuration()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	args := mux.Vars(r)
	spine := args["arg"]
	fmt.Printf("spine = %s\n", spine)

	// File upload will use ::: r.ParseMultipartForm(?? << ??)
	r.ParseForm()
	for k, v := range r.Form {
		fmt.Printf("%s = %s\n", k, v)
	}

	log.Info().
		Str("host", r.Host).
		Str("ref", r.RemoteAddr).
		Str("xref", r.Header.Get("X-Forwarded-For")).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("proto", r.Proto).
		Str("agent", r.Header.Get("User-Agent")).
		Str("response", "200").
		Msg("New Upload")
}

// JSubmit ::: POST Method for JSON sumission.
func JSubmit(w http.ResponseWriter, r *http.Request) {
	msgPostCnt.Add(1)
	msgTimer := prometheus.NewTimer(msgPostDur)
	defer msgTimer.ObserveDuration()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	var subd Submit

	// decode body into struct and pull the value to be encoded
	if err := json.NewDecoder(r.Body).Decode(&subd); err != nil {
		log.Fatal().Err(err).Msg("failed to decode body")
	}
	source := subd.Text
	spine := subd.SpineString

	fmt.Fprintf(w, "Source: %s ::: Spine: %s\n", source, spine)

	// this works:
	mcMeso := make(chan string) // new line...?
	go Mesostic(source, spine, mcMeso)
	showR := <-mcMeso
	fmt.Println(showR)

	log.Info().
		Str("host", r.Host).
		Str("ref", r.RemoteAddr).
		Str("xref", r.Header.Get("X-Forwarded-For")).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("proto", r.Proto).
		Str("agent", r.Header.Get("User-Agent")).
		Str("response", "200").
		Msg("New JSON")
}

// readiness checks are counted but not logged
func ping(w http.ResponseWriter, r *http.Request) {
	pingCnt.Add(1)
	w.Write([]byte("pong\n"))
}

// HTTP frontend for Mesostic API
// This controls flow.
func main() {
	// Prometheus
	prometheus.MustRegister(msgPostCnt)
	prometheus.MustRegister(msgPostDur)
	prometheus.MustRegister(pingCnt)

	rt := mux.NewRouter()
	rt.HandleFunc("/ping", ping)
	rt.Handle("/metrics", promhttp.Handler())

	api := rt.PathPrefix("/app").Subrouter()
	api.HandleFunc("", JSubmit).Methods(http.MethodPost)       // JSON
	api.HandleFunc("/{arg}", FSubmit).Methods(http.MethodPost) // File upload

	if err := http.ListenAndServe(":9999", rt); err != nil {
		log.Fatal().Err(err).Msg("startup failed!")
	}
}
