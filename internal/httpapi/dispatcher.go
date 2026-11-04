package httpapi

import (
	"io/fs"
	"net/http"
	"sync"
)

// Dispatcher keeps the frontend and the setup-only API available while the
// database is not configured, then swaps in the complete API once setup succeeds.
// Requests already in flight keep the handler they started with.
type Dispatcher struct {
	spa      http.Handler
	fallback http.Handler

	mu  sync.RWMutex
	api http.Handler
}

func NewDispatcher(frontendFS fs.FS, fallback http.Handler) *Dispatcher {
	return &Dispatcher{
		spa:      newSPAHandler(frontendFS),
		fallback: fallback,
	}
}

func (d *Dispatcher) Publish(api http.Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.api = api
}

func (d *Dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !isAPIRequest(r.URL.Path) {
		d.spa.ServeHTTP(w, r)
		return
	}

	d.mu.RLock()
	api := d.api
	d.mu.RUnlock()
	if api == nil {
		api = d.fallback
	}
	api.ServeHTTP(w, r)
}
