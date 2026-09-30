package ex2

import (
	"encoding/json"
	"net/http"
	"strconv"
)

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
	mux.HandleFunc("GET /arms", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, stats())
	})
	mux.HandleFunc("GET /arms/{arm}", func(w http.ResponseWriter, r *http.Request) {
		arm, err := strconv.Atoi(r.PathValue("arm"))
		if err != nil {
			http.Error(w, "arm must be a whole number", http.StatusBadRequest)
			return
		}
		all := stats()
		if arm < 0 || arm >= len(all) {
			http.Error(w, "no such arm", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, all[arm])
	})
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
