package twithook

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	twitterscraper "github.com/n0madic/twitter-scraper"
)

func Run(ctx context.Context, user, filter, url, method, data, authUser, authPass string, header http.Header) error {
	top := time.Now().Add(-1 * time.Minute)
	scraper := twitterscraper.New().WithReplies(false)
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	proc := &process{
		scraper:  scraper,
		client:   client,
		url:      url,
		method:   method,
		data:     data,
		authUser: authUser,
		authPass: authPass,
		header:   header,
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(1 * time.Second):
		}
		var err error
		top, err = proc.run(ctx, top, user, filter)
		if err != nil {
			return err
		}
	}
}

type process struct {
	scraper  *twitterscraper.Scraper
	client   *http.Client
	url      string
	method   string
	data     string
	authUser string
	authPass string
	header   http.Header
}

func (p *process) run(ctx context.Context, top time.Time, user, filter string) (time.Time, error) {
	last := top
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for tweet := range p.scraper.GetTweets(ctx, user, 50) {
		if tweet.Error != nil {
			return top, tweet.Error
		}
		// Update most recent tweet timestamp
		if tweet.TimeParsed.After(top) {
			top = tweet.TimeParsed
		}
		// Tweets come ordered by timestamp, stop if is older than last mark
		if !tweet.TimeParsed.After(last) {
			// Pinned tweets aren't ordered by timestamp
			if tweet.IsPin {
				continue
			}
			return top, nil
		}
		if !strings.Contains(strings.ToLower(tweet.Text), filter) {
			continue
		}
		if err := p.webhook(); err != nil {
			return top, err
		}
		log.Printf("tweet: %s, webhook: %s\n", strings.Replace(tweet.Text, "\n", " ", -1), p.url)
	}
	return top, nil
}

func (p *process) webhook() error {
	var body io.Reader
	if p.data != "" {
		body = strings.NewReader(p.data)
	}
	req, err := http.NewRequest(p.method, p.url, body)
	req.Header = p.header
	if p.authUser != "" {
		req.SetBasicAuth(p.authUser, p.authPass)
	}
	if err != nil {
		return fmt.Errorf("couldn't create request: %w", err)
	}
	res, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("invalid status code: %s", res.Status)
	}
	return nil
}
