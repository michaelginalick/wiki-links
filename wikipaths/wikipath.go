package wikipath

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"sync/atomic"
	"wikipaths/links"

	"golang.org/x/exp/slog"
)

const (
	DefaultThreadCount int    = 3
	WikipediaHost      string = "en.wikipedia.org"
	MaxThreads         int    = 10
	MinThreads         int    = 0
)

type Application struct {
	TotalCount  uint64
	Source      *url.URL
	Sink        *url.URL
	Worklist    chan ([]*url.URL)
	UnseenLinks chan (*url.URL)
	SeenLinks   map[string]bool
	ThreadCount int
	WikiClient  *wikiClient
	LinkClient  links.Client
}

type Client interface {
	Crawl(url string) []string
}

type wikiClient struct {
	client *Client
}

func newWikiClient(client *Client) *wikiClient {
	return &wikiClient{
		client: client,
	}
}

func WithSourceLink(link string) func(*Application) error {
	return func(app *Application) error {
		url, err := url.Parse(link)
		if err != nil {
			return err
		}

		if url.Host != WikipediaHost {
			return errors.New("must be a wikipedia link")
		}

		app.Source = url
		return nil
	}
}

func WithSinkLink(link string) func(*Application) error {
	return func(app *Application) error {
		url, err := url.Parse(link)
		if err != nil {
			return err
		}

		if url.Host != WikipediaHost {
			return errors.New("must be a wikipedia link")
		}

		app.Sink = url
		return nil
	}
}

func WithThreadCount(num int) func(*Application) error {
	return func(app *Application) error {
		if num > MaxThreads || num <= MinThreads {
			return fmt.Errorf("thread count must be between %d and %d", MinThreads, MaxThreads)
		}
		app.ThreadCount = num
		return nil
	}
}

type Option func(*Application) error

func New(opts ...Option) (*Application, error) {
	var wikiClient *Client

	app := &Application{
		Worklist:    make(chan []*url.URL),
		UnseenLinks: make(chan *url.URL),
		SeenLinks:   make(map[string]bool),
		WikiClient:  newWikiClient(wikiClient),
		LinkClient:  links.DefaultLinkClient(),
		ThreadCount: DefaultThreadCount,
	}

	for _, opt := range opts {
		if err := opt(app); err != nil {
			return &Application{}, fmt.Errorf("option failed %w", err)
		}
	}

	return app, nil
}

func (app *Application) Run() {
	go func() { app.Worklist <- []*url.URL{app.Source} }()

	// Limit active thread count to app.ThreadCount; default is 3
	for i := 0; i < app.ThreadCount; i++ {
		go func() {
			for link := range app.UnseenLinks {
				foundLinks := app.Crawl(link)
				go func() { app.Worklist <- foundLinks }()
			}
		}()
	}

	for list := range app.Worklist {
		for _, url := range list {
			urlStr := url.String()
			if !app.SeenLinks[urlStr] {
				app.SeenLinks[urlStr] = true
				if urlStr == app.Sink.String() {
					slog.Info("FOUND THE LINK IN", slog.Uint64("count", app.TotalCount))
					os.Exit(0)
				} else {
					atomic.AddUint64(&app.TotalCount, 1)
					slog.Info("Count at", slog.Uint64("count", app.TotalCount))
					app.UnseenLinks <- url
				}
			}
		}
	}
}

func (app *Application) Crawl(url *url.URL) []*url.URL {
	slog.Info("Crawling", slog.String("url", url.String()))
	list, err := app.LinkClient.ExtractWikiLinks(url.String(), WikipediaHost)
	if err != nil {
		slog.Error("error parsing link", err)
	}
	return list
}
