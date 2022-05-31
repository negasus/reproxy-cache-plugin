package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/umputun/go-flags"
	"github.com/umputun/reproxy/lib"
)

var version = "unknown"

var opts struct {
	Listen         string `short:"l" long:"listen" env:"LISTEN" description:"listen on host:port" default:"0.0.0.0:8080"`
	ReproxyAddress string `short:"r" long:"reproxy" env:"REPROXY" description:"reproxy plugins endpoint" default:"http://127.0.0.1:8081"`
}

type cacheItem struct {
	deadline time.Time
	body     []byte
	headers  http.Header
}

var (
	mx    *sync.RWMutex
	cache map[string]*cacheItem
)

func main() {
	fmt.Printf("reproxy-cache-plugin %s\n", version)

	p := flags.NewParser(&opts, flags.PrintErrors|flags.PassDoubleDash|flags.HelpFlag)
	p.SubcommandsOptional = true
	if _, err := p.Parse(); err != nil {
		if err.(*flags.Error).Type != flags.ErrHelp {
			log.Printf("[ERROR] cli error: %v", err)
		}
		os.Exit(2)
	}

	log.Printf("options: %#v", opts)

	mx = &sync.RWMutex{}
	cache = make(map[string]*cacheItem)

	err := run()
	if err != nil {
		log.Printf("run plugin failed, %v", err)
		os.Exit(1)
	}

	log.Printf("done")
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := &Handler{}

	go func() {
		if x := recover(); x != nil {
			panic(x)
		}
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, os.Kill)
		<-stop
		cancel()
	}()

	plugin := lib.Plugin{
		Name:    "cache",
		Address: opts.Listen,
		Methods: []string{"Before", "After.Tail"},
	}

	return plugin.Do(ctx, opts.ReproxyAddress, h)
}

type Handler struct{}

func (h *Handler) Before(req lib.Request, res *lib.Response) (err error) {
	fmt.Printf(">> before\n")
	if req.Method != http.MethodGet {
		return nil
	}

	mx.RLock()
	defer mx.RUnlock()

	fmt.Printf("find in cache %s\n", req.URL)

	item, ok := cache[req.URL]
	if ok && item.deadline.After(time.Now()) {
		fmt.Printf("find in cache %s - FOUND\n", req.URL)
		res.Body = item.body
		res.StatusCode = http.StatusOK
		res.HeadersOut = item.headers
		res.Break = true
		return nil

	}

	fmt.Printf("find in cache %s - NOT FOUND\n", req.URL)

	return nil
}

func (h *Handler) After(req lib.Request, res *lib.Response) (err error) {
	fmt.Printf(">> after\n")
	if req.Method != http.MethodGet {
		return nil
	}

	mx.Lock()
	defer mx.Unlock()

	fmt.Printf("add to cache %s\n", req.URL)

	cache[req.URL] = &cacheItem{
		deadline: time.Now().Add(time.Second * 5),
		body:     res.Body,
		headers:  res.HeadersOut,
	}

	res.StatusCode = req.ResponseCode
	res.Body = req.ResponseBody
	res.HeadersOut = req.ResponseHeaders

	return
}
