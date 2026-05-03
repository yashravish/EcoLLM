package pool

import (
	"net/http"
	"sync"
	"time"
)

// Pool maintains one reusable http.Client per upstream base URL so that
// TCP connections are shared across requests to the same vLLM instance.
type Pool struct {
	mu      sync.RWMutex
	clients map[string]*http.Client
}

func New() *Pool {
	return &Pool{clients: make(map[string]*http.Client)}
}

// GetOrCreate returns the existing client for baseURL, or creates one with
// sensible transport defaults tuned for long-running inference requests.
func (p *Pool) GetOrCreate(baseURL string) *http.Client {
	p.mu.RLock()
	c, ok := p.clients[baseURL]
	p.mu.RUnlock()
	if ok {
		return c
	}

	c = &http.Client{
		Timeout: 120 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        50,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	p.mu.Lock()
	p.clients[baseURL] = c
	p.mu.Unlock()
	return c
}