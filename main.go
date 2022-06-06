package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/go-flags"
	"github.com/umputun/reproxy/lib"
)

var version = "unknown"

/*
Rule format:
<path> <methods> <timeout> [<cache errors>]

<path> 			- route (request.Route value)
<methods> 		- http methods, | separated
<timeout> 		- time duration
<cache errors> 	- string 'yes' or omit

http://127.0.0.1:2000/foo GET 60s
http://127.0.0.1:2000/foo GET|POST 60s yes
*/

var opts struct {
	Listen         string   `short:"l" long:"listen" env:"LISTEN" description:"listen on host:port" default:"0.0.0.0:8080"`
	ReproxyAddress string   `short:"a" long:"reproxy" env:"REPROXY" description:"reproxy plugins endpoint" default:"http://127.0.0.1:8081"`
	Rules          []string `short:"r" long:"rules" env:"RULES" description:"cache rules"`
	Dbg            bool     `long:"dbg" env:"DEBUG" description:"debug mode"`
}

type rule struct {
	methods     map[string]struct{}
	timeout     time.Duration
	cacheErrors bool
}

type storage interface {
	Get(key string) (*item, error)
	Put(key string, i *item) error
}

type Handler struct {
	storage storage
	rules   map[string]rule
}

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

	setupLog(opts.Dbg)

	log.Printf("[INFO] options: %#v", opts)

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

	h := &Handler{
		storage: newStorageMemory(ctx),
		rules:   map[string]rule{},
	}

	errParseRules := h.parseRules(opts.Rules)
	if errParseRules != nil {
		log.Printf("[ERROR] error parse rules: %v", errParseRules)
		os.Exit(2)
	}

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
		Name:        "cache",
		Address:     opts.Listen,
		Methods:     []string{"Before"},
		TailMethods: []string{"After"},
	}

	return plugin.Do(ctx, opts.ReproxyAddress, h)
}

func setupLog(dbg bool) {
	if dbg {
		log.Setup(log.Debug, log.CallerFile, log.CallerFunc, log.Msec, log.LevelBraces)
		return
	}
	log.Setup(log.Msec, log.LevelBraces)
}
