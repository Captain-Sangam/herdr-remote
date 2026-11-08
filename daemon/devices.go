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
	path string
	mu   sync.RWMutex
	list []Device
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
		if s.list[i].SPKI == key {
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

func (s *deviceStore) remove(id string) (Device, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, x := range s.list {
		if x.ID == id {
			s.list = append(s.list[:i:i], s.list[i+1:]...)
			return x, true, s.saveLocked()
		}
	}
	return Device{}, false, nil
}

func (s *deviceStore) saveLocked() error {
	b, err := json.MarshalIndent(s.list, "", " ")
	if err != nil {
		return err
	}
	return writeFileAtomic(s.path, append(b, '\n'), 0o600)
}
