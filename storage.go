package main

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"
)

var (
	ErrNotFound = errors.New("not found")
)

type item struct {
	deadline time.Time
	status   int
	body     []byte
	headers  http.Header
}

type storageMemory struct {
	mx   *sync.RWMutex
	data map[string]*item
}

func newStorageMemory(ctx context.Context) *storageMemory {
	s := &storageMemory{
		mx:   &sync.RWMutex{},
		data: map[string]*item{},
	}

	go s.cleanup(ctx)

	return s
}

func (s *storageMemory) cleanup(ctx context.Context) {
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.mx.Lock()
			now := time.Now()
			for key, i := range s.data {
				if i.deadline.Before(now) {
					delete(s.data, key)
				}
			}
			s.mx.Unlock()
		}
	}
}

func (s *storageMemory) Get(key string) (*item, error) {
	s.mx.Lock()
	defer s.mx.Unlock()

	i, ok := s.data[key]
	if !ok {
		return nil, ErrNotFound
	}

	if i.deadline.Before(time.Now()) {
		delete(s.data, key)
		return nil, ErrNotFound
	}

	return i, nil
}
func (s *storageMemory) Put(key string, i *item) error {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.data[key] = i

	return nil
}
