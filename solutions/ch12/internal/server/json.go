package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// maxBody is the largest request body the server will read.
const maxBody = 1 << 20 // 1 MiB

// errorBody is the JSON shape of every error response.
type errorBody struct {
	Error string `json:"error"`
}

// writeJSON sends v as the response body with the given status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Once the header is out, an encoding failure (usually a client that hung
	// up) cannot change the status, so there is nothing useful to do with it.
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorBody{Error: msg})
}

// decodeJSON reads exactly one JSON value from the request body into dst. It
// rejects unknown fields, trailing data, and bodies over maxBody.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return errors.New("body must contain a single JSON value")
	}
	return nil
}

// badBody writes the right error response for a decodeJSON failure.
func badBody(w http.ResponseWriter, err error) {
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
		return
	}
	writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
}
