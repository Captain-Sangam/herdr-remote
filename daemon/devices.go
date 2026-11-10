package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

type Device struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	SPKI  string `json:"spki_sha256"`
	Added string `json:"added"`
}

type deviceStore struct {
	path    string
	mu      sync.RWMutex
	list    []Device
	revoked map[string]bool // ids whose revoke is not yet written to the file; their keys are unknown
}

func openDevices(path string) (*deviceStore, error) {
	s := &deviceStore{path: path}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &s.list); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *deviceStore) all() []Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Device(nil), s.list...)
}

func (s *deviceStore) bySPKI(h [32]byte) *Device {
	key := b64url(h[:])
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.list {
		if s.list[i].SPKI == key && !s.revoked[s.list[i].ID] {
			d := s.list[i]
			return &d
		}
	}
	return nil
}

func (s *deviceStore) byID(id string) *Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.list {
		if s.list[i].ID == id {
			d := s.list[i]
			return &d
		}
	}
	return nil
}

// add stores a device. A key that is already paired replaces its old entry.
func (s *deviceStore) add(name string, h [32]byte) (Device, error) {
	var idb [8]byte
	if _, err := rand.Read(idb[:]); err != nil {
		return Device{}, err
	}
	d := Device{ID: hex.EncodeToString(idb[:]), Name: name, SPKI: b64url(h[:]), Added: time.Now().UTC().Format(time.RFC3339)}
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.list[:0:0]
	for _, x := range s.list {
		if x.SPKI != d.SPKI {
			kept = append(kept, x)
		}
	}
	s.list = append(kept, d)
	return d, s.saveLocked()
}

// mark marks a device as revoked: from now on its key is unknown. It stays in
// the list, so a retry after a failed write finds it, until drop writes the file.
func (s *deviceStore) mark(id string) (Device, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, x := range s.list {
		if x.ID == id {
			if s.revoked == nil {
				s.revoked = map[string]bool{}
			}
			s.revoked[id] = true
			return x, true
		}
	}
	return Device{}, false
}

// drop writes the file without the device, then forgets it. A failed write
// keeps the device in the list with its mark, so a retry writes again.
func (s *deviceStore) drop(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.list
	s.list = old[:0:0]
	for _, x := range old {
		if x.ID != id {
			s.list = append(s.list, x)
		}
	}
	if err := s.saveLocked(); err != nil {
		s.list = old
		return err
	}
	delete(s.revoked, id)
	return nil
}

func (s *deviceStore) saveLocked() error {
	b, err := json.MarshalIndent(s.list, "", " ")
	if err != nil {
		return err
	}
	return writeFileAtomic(s.path, append(b, '\n'), 0o600)
}
