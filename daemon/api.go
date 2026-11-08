package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func apiError(w http.ResponseWriter, status int, msg, reason string) {
	writeJSON(w, status, errBody(msg, reason))
}

func (d *Daemon) routes() {
	d.mux.HandleFunc("GET /v1/hello", d.handleHello)
	d.mux.HandleFunc("GET /v1/spaces", d.handleSpaces)
	d.mux.HandleFunc("GET /v1/agents", d.handleAgents)
	d.mux.HandleFunc("GET /v1/terminals/{t}", d.handleTerminal)
	d.mux.HandleFunc("GET /v1/events", d.handleEvents)
	d.mux.HandleFunc("GET /v1/terminals/{t}/live", d.handleLive)
	d.mux.HandleFunc("GET /v1/terminals/{t}/transcript", d.handleTranscript)
	d.mux.HandleFunc("GET /v1/terminals/{t}/prompt", d.handlePrompt)
	d.mux.HandleFunc("POST /v1/terminals/{t}/decision", d.handleDecision)
	d.mux.HandleFunc("POST /v1/terminals/{t}/message", d.handleMessage)
}

func (d *Daemon) handleHello(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{
		"daemon_version": version,
		"server_time_ms": time.Now().UnixMilli(),
		"endpoints":      d.endpoints(),
	})
}

func (d *Daemon) handleSpaces(w http.ResponseWriter, r *http.Request) {
	if !d.state.isOK() {
		apiError(w, 503, "hc unavailable", "")
		return
	}
	writeJSON(w, 200, d.state.spaces(r.Context()))
}

func (d *Daemon) handleAgents(w http.ResponseWriter, r *http.Request) {
	if !d.state.isOK() {
		apiError(w, 503, "hc unavailable", "")
		return
	}
	writeJSON(w, 200, d.state.agents())
}

// agentTarget resolves {t} to a live reporting local terminal, or writes the error.
func (d *Daemon) agentTarget(w http.ResponseWriter, r *http.Request) (*Terminal, bool) {
	t := r.PathValue("t")
	if !validTarget(t) {
		apiError(w, 400, "bad target", "")
		return nil, false
	}
	if !d.state.isOK() {
		apiError(w, 503, "hc unavailable", "")
		return nil, false
	}
	term := d.state.terminal(t)
	if term == nil {
		apiError(w, 404, "no such terminal", "")
		return nil, false
	}
	if !isAgent(term) {
		apiError(w, 409, "not an agent", "not_agent")
		return nil, false
	}
	return term, true
}

func (d *Daemon) handleTerminal(w http.ResponseWriter, r *http.Request) {
	term, ok := d.agentTarget(w, r)
	if !ok {
		return
	}
	d.state.mu.RLock()
	a := d.state.agentLocked(term)
	d.state.mu.RUnlock()
	b, _ := json.Marshal(a)
	var m map[string]any
	json.Unmarshal(b, &m)
	m["cols"], m["rows"] = term.Cols, term.Rows
	writeJSON(w, 200, m)
}

func (d *Daemon) handleEvents(w http.ResponseWriter, r *http.Request) {
	sub := d.state.subscribe()
	if sub == nil {
		apiError(w, 503, "hc unavailable", "")
		return
	}
	defer d.state.unsubscribe(sub)
	fl, ok := w.(http.Flusher)
	if !ok {
		apiError(w, 500, "no streaming", "")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(200)
	fl.Flush()
	ka := time.NewTicker(d.state.kaEvery)
	defer ka.Stop()
	for {
		select {
		case ev := <-sub.ch:
			b, err := json.Marshal(ev.data)
			if err != nil {
				return
			}
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.event, b); err != nil {
				return
			}
			fl.Flush()
		case <-ka.C:
			if _, err := fmt.Fprint(w, ": ka\n\n"); err != nil {
				return
			}
			fl.Flush()
		case <-sub.closed:
			return
		case <-r.Context().Done():
			return
		}
	}
}
