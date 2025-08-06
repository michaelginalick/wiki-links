package links

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/exp/slog"
	"golang.org/x/net/html"
)

type Client interface {
	ExtractWikiLinks(url, scopedHost string) ([]*url.URL, error)
}

type linkClient struct {
	httpClient *http.Client
}

// DefaultLinkClient returns a Client using http.DefaultClient.
func DefaultLinkClient() Client {
	return NewLinkClient(http.DefaultClient)
}

// NewLinkClient returns a Client using a custom HTTP client.
func NewLinkClient(httpClient *http.Client) Client {
	return &linkClient{
		httpClient: httpClient,
	}
}

func (link *linkClient) ExtractWikiLinks(givenURL, scopedHost string) ([]*url.URL, error) {
	doc, err := fetchHTLMFromLink(link.httpClient, givenURL)
	if err != nil || doc == nil {
		return nil, fmt.Errorf("could not fetch html from link %w", err)
	}

	baseURL, err := url.Parse(givenURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	urls := []*url.URL{}
	visitNode := func(n *html.Node) {
		if n != nil && n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {

				if a.Key != "href" {
					continue
				}

				url, err := url.Parse(a.Val)
				if err != nil {
					slog.Error("parsing a href tag into url")
					continue
				}

				fullURL := baseURL.ResolveReference(url)

				// ignore bad, non-wikipedia, and urls with a query param
				if fullURL.Host != "" && fullURL.Host == scopedHost && len(fullURL.RawQuery) == 0 {
					urls = append(urls, fullURL)
				}
			}
		}
	}

	forEachNode(doc, visitNode, nil)
	return urls, nil
}

func fetchHTLMFromLink(client *http.Client, url string) (*html.Node, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(2)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

// forEachNode recursively visits HTML nodes using the pre/post functions.
func forEachNode(n *html.Node, pre, post func(n *html.Node)) {
	if pre != nil {
		pre(n)
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		forEachNode(c, pre, post)
	}

	if post != nil {
		post(n)
	}
}
