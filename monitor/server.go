package monitor

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func StartServer(port string) {
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":"+port, nil)
}
