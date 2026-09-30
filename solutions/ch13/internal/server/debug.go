package server

import (
	"net/http"
	"net/http/pprof"
)

// DebugHandler serves the runtime's profiling endpoints under /debug/pprof/.
//
// Profiles expose internals (source paths, memory contents, goroutine stacks)
// and let a caller make the process do expensive work. Serve this handler on a
// separate listener bound to localhost or an internal network, never on the
// public port.
//
// Importing net/http/pprof for its side effects registers these routes on
// http.DefaultServeMux, which is easy to expose by accident. Registering them
// by hand on a mux of our own keeps them off the main server.
func DebugHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index) // also serves heap, goroutine, mutex, block, allocs
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile) // CPU profile
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return mux
}
