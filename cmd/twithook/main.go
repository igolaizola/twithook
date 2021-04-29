package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/igolaizola/twithook"
)

func main() {
	// Parse flags
	user := flag.String("user", "", "twitter username to get tweets from")
	filter := flag.String("filter", "", "keywords to search")
	url := flag.String("url", "", "webhook url")
	method := flag.String("method", "GET", "webhook http method (GET, POST...)")
	data := flag.String("data", "", "webhook post data")
	authUser := flag.String("auth-user", "", "basic auth user")
	authPass := flag.String("auth-pass", "", "basic auth pass")
	header := make(http.Header)
	headerVal := &headerValue{header: header}
	flag.Var(headerVal, "header", "http header with format header:value")

	flag.Parse()
	if *user == "" {
		log.Fatal("user not provided")
	}
	if *filter == "" {
		log.Fatal("filter not provided")
	}
	if *url == "" {
		log.Fatal("url not provided")
	}

	// Create signal based context
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
			cancel()
		}
		signal.Stop(c)
	}()

	// Run bot
	if err := twithook.Run(ctx, *user, *filter, *url, *method, *data, *authUser, *authPass, header); err != nil {
		log.Fatal(err)
	}
}

// headerValue is a flags.Value implementation for http.Header
type headerValue struct {
	header http.Header
}

func (h *headerValue) String() string {
	return fmt.Sprintf("%v", h.header)
}

func (h *headerValue) Set(value string) error {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("could not parse header value %s", value)
	}
	k, v := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	h.header.Add(k, v)
	return nil
}
