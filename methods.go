package main

import (
	"errors"
	"time"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/reproxy/lib"
)

func (h *Handler) Before(req lib.Request, res *lib.Response) error {
	key := req.Method + "@" + req.Route

	// fast path, if rule is not registered, do not touch cache
	r, ok := h.rules[req.Route]
	if !ok {
		log.Printf("[DEBUG] rule for %q is not registered", req.Route)
		return nil
	}
	if _, ok := r.methods[req.Method]; !ok {
		log.Printf("[DEBUG] method %q for %q is not registered", req.Method, req.Route)
		return nil
	}

	i, err := h.storage.Get(key)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			log.Printf("[DEBUG] cache for %q not found", key)
			return nil
		}
		log.Printf("[ERROR] error get from cache: %v", err)
		return err
	}

	log.Printf("[DEBUG] response %q from cache", key)

	res.Break = true
	res.Body = i.body
	res.StatusCode = i.status
	res.HeadersOut = i.headers
	res.OverrideHeadersOut = true

	return nil
}

func (h *Handler) After(req lib.Request, _ *lib.Response) error {
	r, ok := h.rules[req.Route]
	if !ok {
		log.Printf("[DEBUG] rule for %q is not registered", req.Route)
		return nil
	}
	if _, ok := r.methods[req.Method]; !ok {
		log.Printf("[DEBUG] method %q for %q is not registered", req.Method, req.Route)
		return nil
	}

	key := req.Method + "@" + req.Route

	log.Printf("[DEBUG] %q cached", key)

	errPut := h.storage.Put(key, &item{
		deadline: time.Now().Add(r.timeout),
		status:   req.ResponseCode,
		body:     req.ResponseBody,
		headers:  req.ResponseHeaders,
	})

	if errPut != nil {
		log.Printf("[ERROR] error put item to cache: %v", errPut)
		return errPut
	}

	return nil
}
