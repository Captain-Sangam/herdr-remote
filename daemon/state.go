package main

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"
)

// agentJSON is the agent shape of section 3.4.
type agentJSON struct {
	Address string `json:"address"`
	Space   string `json:"space"`
	Tab     string `json:"tab"`
	Where   string `json:"where"`
	App     string `json:"app"`
	State   string `json:"state"`
	Kind    string `json:"kind,omitempty"`
	Msg     string `json:"msg,omitempty"`
	Turn    int64  `json:"turn"`
	Title   string `json:"title"`
}

type spaceJSON struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Agents []agentJSON `json:"agents"`
}

// placement is where Herdr shows a terminal, from `herdr-app ls`.
type placement struct {
	spaceID, spaceName, tabID, tabName string
}

type layout struct {
	open   bool // Herdr.app answered
	spaces []struct{ id, name string }
	place  map[string]placement
}

type sseEvent struct {
	event string
	data  any
}

type subscriber struct {
	ch     chan sseEvent
	closed chan struct{}
	once   sync.Once
}

func (s *subscriber) end() { s.once.Do(func() { close(s.closed) }) }

// state mirrors hc's terminals from the events connection.
type state struct {
	d        *Daemon
	mu       sync.RWMutex
	terms    map[string]*Terminal
	ok       bool
	lay      layout
	layAt    time.Time
	subs     map[*subscriber]struct{}
	onSize   func(terminal string) // step 5: end live streams of a resized terminal
	kaEvery  time.Duration
	deadline time.Duration // no note for this long means the connection is dead
	conn     *hcConn       // the current events connection (tests)
}

func newState(d *Daemon) *state {
	return &state{d: d, terms: map[string]*Terminal{}, subs: map[*subscriber]struct{}{}, kaEvery: 10 * time.Second, deadline: 45 * time.Second}
}

func isAgent(t *Terminal) bool {
	return t != nil && t.State == "live" && t.Reported && validTarget(t.Address)
}

func (s *state) isOK() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ok
}

func (s *state) terminal(addr string) *Terminal {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.terms[addr]
	if !ok {
		return nil
	}
	c := *t
	return &c
}

// run keeps one subscription alive, with backoff, until ctx ends.
func (s *state) run(ctx context.Context) {
	backoff := time.Second
	for ctx.Err() == nil {
		snap, conn, err := s.d.hc.Subscribe(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			backoff = min(2*backoff, maxBackoff)
			continue
		}
		backoff = time.Second
		s.mu.Lock()
		s.conn = conn
		s.mu.Unlock()
		s.refreshLayout(ctx)
		s.applySnapshot(snap)
		s.consume(ctx, conn)
		conn.close()
		s.lost()
	}
}

func (s *state) consume(ctx context.Context, conn *hcConn) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-conn.done:
			return
		case <-time.After(s.deadline):
			return
		case n := <-conn.notes:
			s.handle(ctx, n)
		}
	}
}

func (s *state) applySnapshot(snap *Snapshot) {
	s.mu.Lock()
	s.terms = map[string]*Terminal{}
	for i := range snap.Terminals {
		t := snap.Terminals[i]
		s.terms[t.Address] = &t
	}
	s.ok = true
	agents := s.agentsLocked()
	s.mu.Unlock()
	s.broadcast(sseEvent{"snapshot", agents})
}

func (s *state) lost() {
	s.mu.Lock()
	s.ok = false
	subs := s.subs
	s.subs = map[*subscriber]struct{}{}
	s.mu.Unlock()
	for sub := range subs {
		sub.end()
	}
}

func (s *state) handle(ctx context.Context, n Note) {
	switch n.Method {
	case "snapshot":
		s.refreshLayout(ctx)
		s.applySnapshot(n.Snapshot)
		return
	case "event":
	default:
		return
	}
	s.mu.Lock()
	t := s.terms[n.Terminal]
	wasAgent := isAgent(t)
	refresh := false
	switch n.Type {
	case "created":
		var term Terminal
		if json.Unmarshal(n.Data, &term) == nil && term.Address != "" {
			s.terms[term.Address] = &term
			t = &term
		}
		refresh = true
	case "status":
		var d statusData
		if t != nil && json.Unmarshal(n.Data, &d) == nil {
			t.Reported = d.Reported
			t.Status = d.Status
		}
	case "turn":
		var d turnData
		if t != nil && json.Unmarshal(n.Data, &d) == nil {
			t.Turn = d.Turn
		}
	case "title":
		var d titleData
		if t != nil && json.Unmarshal(n.Data, &d) == nil {
			t.Title = d.Title
		}
	case "labels":
		var d labelsData
		if t != nil && json.Unmarshal(n.Data, &d) == nil {
			t.Labels = d.Labels
		}
		refresh = true
	case "size":
		var d sizeData
		if t != nil && json.Unmarshal(n.Data, &d) == nil {
			t.Cols, t.Rows = d.Cols, d.Rows
		}
		if s.onSize != nil {
			go s.onSize(n.Terminal)
		}
	case "foreground":
		var fg Foreground
		if t != nil && json.Unmarshal(n.Data, &fg) == nil {
			t.Foreground = &fg
		}
	case "finished":
		if t != nil {
			t.State = "finished"
		}
		refresh = true
	case "removed":
		delete(s.terms, n.Terminal)
		t = nil
		refresh = true
	default:
		// lost, restarted, keep, doc and anything new: refetch the facts.
		s.mu.Unlock()
		s.refetch(ctx, n.Terminal)
		return
	}
	var ev *sseEvent
	switch {
	case isAgent(t):
		a := s.agentLocked(t)
		ev = &sseEvent{"agent", a}
	case wasAgent:
		ev = &sseEvent{"agent_removed", map[string]string{"address": n.Terminal}}
	}
	s.mu.Unlock()
	if refresh {
		s.refreshLayout(ctx)
		// The placement may have changed: resend the agent with its new space.
		if t != nil && isAgent(t) {
			s.mu.RLock()
			if cur := s.terms[n.Terminal]; cur != nil {
				ev = &sseEvent{"agent", s.agentLocked(cur)}
			}
			s.mu.RUnlock()
		}
	}
	if ev != nil {
		s.broadcast(*ev)
	}
}

func (s *state) refetch(ctx context.Context, addr string) {
	if !validTarget(addr) {
		return
	}
	term, err := s.d.hc.Get(ctx, addr)
	s.mu.Lock()
	old := s.terms[addr]
	wasAgent := isAgent(old)
	if err == nil {
		s.terms[addr] = term
	} else if err == errNotFound {
		delete(s.terms, addr)
		term = nil
	} else {
		term = old
	}
	var ev *sseEvent
	switch {
	case isAgent(term):
		ev = &sseEvent{"agent", s.agentLocked(term)}
	case wasAgent:
		ev = &sseEvent{"agent_removed", map[string]string{"address": addr}}
	}
	s.mu.Unlock()
	if ev != nil {
		s.broadcast(*ev)
	}
}

// refreshLayout runs `herdr-app ls` and caches the result.
func (s *state) refreshLayout(ctx context.Context) layout {
	lay := layout{place: map[string]placement{}}
	if out, err := s.d.r.herdrAppLs(ctx); err == nil {
		var ls struct {
			Spaces []struct {
				ID      string  `json:"id"`
				Name    string  `json:"name"`
				Machine *string `json:"machine"`
				Tabs    []struct {
					ID        string `json:"id"`
					Name      string `json:"name"`
					Terminals []struct {
						Terminal  string   `json:"terminal"`
						SubAgents []string `json:"sub_agents"` // addresses of hidden children shown under it
					} `json:"terminals"`
				} `json:"tabs"`
			} `json:"spaces"`
		}
		if json.Unmarshal(out, &ls) == nil {
			lay.open = true
			for _, sp := range ls.Spaces {
				if sp.Machine != nil && *sp.Machine != "" {
					continue
				}
				lay.spaces = append(lay.spaces, struct{ id, name string }{sp.ID, sp.Name})
				for _, tab := range sp.Tabs {
					for _, term := range tab.Terminals {
						p := placement{sp.ID, sp.Name, tab.ID, tab.Name}
						lay.place[term.Terminal] = p
						for _, sub := range term.SubAgents {
							lay.place[sub] = p
						}
					}
				}
			}
		}
	}
	s.mu.Lock()
	s.lay, s.layAt = lay, time.Now()
	s.mu.Unlock()
	return lay
}

func splitWhere(where string) (space, tab string) {
	if i := strings.Index(where, " / "); i >= 0 {
		return where[:i], where[i+3:]
	}
	return where, ""
}

func (s *state) agentLocked(t *Terminal) agentJSON {
	a := agentJSON{Address: t.Address, Where: t.Labels["where"], Turn: t.Turn, Title: t.Title}
	if t.Status != nil {
		a.App, a.State = t.Status.App, t.Status.State
		if a.State == "blocked" {
			a.Kind, a.Msg = t.Status.Kind, t.Status.Msg
		}
	}
	if p, ok := s.lay.place[t.Address]; ok {
		a.Space, a.Tab = p.spaceName, p.tabName
	} else if !s.lay.open {
		a.Space, a.Tab = splitWhere(a.Where)
	} else if id := t.Labels["herdr.space"]; id != "" {
		a.Space = id
		for _, sp := range s.lay.spaces {
			if sp.id == id {
				a.Space = sp.name
			}
		}
	}
	return a
}

func (s *state) agentsLocked() []agentJSON {
	var out []agentJSON
	for _, t := range s.terms {
		if isAgent(t) {
			out = append(out, s.agentLocked(t))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Address < out[j].Address })
	if out == nil {
		out = []agentJSON{}
	}
	return out
}

func (s *state) agents() []agentJSON {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.agentsLocked()
}

// spaces groups the agents by Herdr space, with a fresh `herdr-app ls`.
func (s *state) spaces(ctx context.Context) []spaceJSON {
	lay := s.refreshLayout(ctx)
	s.mu.RLock()
	agents := s.agentsLocked()
	s.mu.RUnlock()
	if !lay.open {
		sort.SliceStable(agents, func(i, j int) bool { return agents[i].Where < agents[j].Where })
		return []spaceJSON{{ID: "closed", Name: "Herdr window closed", Agents: agents}}
	}
	byID := map[string]*spaceJSON{}
	var out []spaceJSON
	for _, sp := range lay.spaces {
		out = append(out, spaceJSON{ID: sp.id, Name: sp.name, Agents: []agentJSON{}})
	}
	for i := range out {
		byID[out[i].ID] = &out[i]
	}
	var other []agentJSON
	s.mu.RLock()
	for _, a := range agents {
		t := s.terms[a.Address]
		id := ""
		if p, ok := lay.place[a.Address]; ok {
			id = p.spaceID
		} else if t != nil {
			id = t.Labels["herdr.space"]
		}
		if sp := byID[id]; sp != nil {
			sp.Agents = append(sp.Agents, a)
		} else {
			other = append(other, a)
		}
	}
	s.mu.RUnlock()
	if len(other) > 0 {
		out = append(out, spaceJSON{ID: "other", Name: "Not in a space", Agents: other})
	}
	if out == nil {
		out = []spaceJSON{}
	}
	return out
}

func (s *state) subscribe() *subscriber {
	sub := &subscriber{ch: make(chan sseEvent, 64), closed: make(chan struct{})}
	s.mu.Lock()
	if !s.ok {
		s.mu.Unlock()
		return nil
	}
	s.subs[sub] = struct{}{}
	sub.ch <- sseEvent{"snapshot", s.agentsLocked()}
	s.mu.Unlock()
	return sub
}

func (s *state) unsubscribe(sub *subscriber) {
	s.mu.Lock()
	delete(s.subs, sub)
	s.mu.Unlock()
	sub.end()
}

// broadcast never blocks: a subscriber that cannot keep up is ended and reconnects.
func (s *state) broadcast(ev sseEvent) {
	s.mu.RLock()
	var slow []*subscriber
	for sub := range s.subs {
		select {
		case sub.ch <- ev:
		default:
			slow = append(slow, sub)
		}
	}
	s.mu.RUnlock()
	for _, sub := range slow {
		s.unsubscribe(sub)
	}
}
