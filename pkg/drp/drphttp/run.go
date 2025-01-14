package drphttp

import (
	"linkany/pkg/drp"
	"net/http"
)

// Start a http server
func Start(opts drp.Options) error {
	drpServer := NewDrpServer()
	//if opts.RunDrp {
	http.HandleFunc("/drp", upgrade(drpServer))
	//}

	return http.ListenAndServe(":8080", nil)
}
