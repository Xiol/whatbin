package whatbin

import (
	"errors"
	"net/http"
)

var ErrNoBinsToday = errors.New("no bins out today")

var ImpersonationHeaders = http.Header{
	"User-Agent":      []string{"Mozilla/5.0 (X11; Fedora; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.111 Safari/537.36"},
	"Accept":          []string{"text/html", "application/xhtml+xml", "application/xml"},
	"Accept-Encoding": []string{"gzip", "deflate", "sdch"},
	"Cache-Control":   []string{"no-cache"},
	"DNT":             []string{"1"},
}

type Provider interface {
	Bins() ([]string, error)
}

type Notifier interface {
	Notify([]string) error
}

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

func Handle(p Provider, n Notifier) error {
	collections, err := p.Bins()
	if err != nil {
		if errors.Is(err, ErrNoBinsToday) {
			return nil
		}
		return err
	}

	if err := n.Notify(collections); err != nil {
		return err
	}
	return nil
}
