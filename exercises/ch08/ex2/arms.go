package ex2

import "net/http"

// ArmStat is one arm's summary.
type ArmStat struct {
	Arm   int     `json:"arm"`
	Pulls int     `json:"pulls"`
	Mean  float64 `json:"mean"`
}

// Routes returns a handler with two routes, using Go 1.22 ServeMux patterns:
//
//	GET /arms          the whole slice returned by stats(), as a JSON array
//	GET /arms/{arm}    one element, as a JSON object
//
// For /arms/{arm}, read the wildcard with r.PathValue("arm"). If it is not a
// whole number respond 400; if it is a number outside the slice respond 404.
// Use the status codes only, with any JSON or plain-text body you like. Other
// methods on these paths should get 405 automatically from the mux.
func Routes(stats func() []ArmStat) http.Handler {
	mux := http.NewServeMux()
	// TODO
	return mux
}
