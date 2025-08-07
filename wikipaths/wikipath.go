package wikipath

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
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
	sync.Mutex
}

type Client interface {
	Crawl(ctx context.Context, url string) []string
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

// Limit active thread count to app.ThreadCount; default is 3
func (app *Application) startWorkers(wg *sync.WaitGroup, ctx context.Context) {
	for i := 0; i < app.ThreadCount; i++ {
		go func() {
			for link := range app.UnseenLinks {
				wg.Add(1)
				foundLinks := app.Crawl(ctx, link)
				go func() {
					defer wg.Done()
					select {
					case <-ctx.Done():
						return
					case app.Worklist <- foundLinks:
					}
				}()
			}
		}()
	}
}

func (app *Application) processWorkList(cancel context.CancelFunc) {
	appSink := app.Sink.String()

	for list := range app.Worklist {
		newPage := false
		for _, url := range list {
			urlStr := url.String()
			seen := app.seen(urlStr)
			if seen {
				continue
			}

			if urlStr == appSink {
				slog.Info("FOUND THE LINK IN", slog.Uint64("count", app.TotalCount))
				cancel()
				return
			}
			app.UnseenLinks <- url
			newPage = true
		}

		if newPage {
			atomic.AddUint64(&app.TotalCount, 1)
			slog.Info("Count at", slog.Uint64("count", app.TotalCount))
		}
	}
}

func (app *Application) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { app.Worklist <- []*url.URL{app.Source} }()
	// start workers
	app.startWorkers(&wg, ctx)
	// process found links
	app.processWorkList(cancel)

	go func() {
		wg.Wait()
		close(app.UnseenLinks)
		close(app.Worklist)
	}()
}

func (app *Application) Crawl(ctx context.Context, url *url.URL) []*url.URL {
	slog.Info("Crawling", slog.String("url", url.String()))
	list, err := app.LinkClient.ExtractWikiLinks(url.String(), WikipediaHost)
	if err != nil {
		slog.Error("error parsing link", err.Error())
	}
	return list
}

func (app *Application) seen(urlStr string) bool {
	app.Lock()
	defer app.Unlock()

	if app.SeenLinks[urlStr] {
		return true
	}
	app.SeenLinks[urlStr] = true
	return false
}
