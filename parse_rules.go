package main

import (
	"fmt"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
)

func (h *Handler) parseRules(ss []string) error {
	for _, s := range ss {
		r := rule{
			methods: map[string]struct{}{},
		}

		parts := strings.Split(s, " ")
		if len(parts) != 3 && len(parts) != 4 {
			return fmt.Errorf("rule must contains 3 or 4 parts, space separated")
		}

		key := parts[0]

		for _, m := range strings.Split(parts[1], "|") {
			r.methods[m] = struct{}{}
		}

		t, err := time.ParseDuration(parts[2])
		if err != nil {
			return fmt.Errorf("error parse rule timeout %q: %v", parts[2], err)
		}
		r.timeout = t

		if len(parts) == 4 && strings.ToLower(parts[3]) == "yes" {
			r.cacheErrors = true
		}

		log.Printf("[INFO] register rule %q", key)

		h.rules[key] = r
	}
	return nil
}
