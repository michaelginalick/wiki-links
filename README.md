# WikiPath

**WikiPath** is a simple, concurrent command-line tool written in Go that calculates the link distance between two Wikipedia pages. It crawls Wikipedia pages by following internal links until it finds the target page.

## What It Does

WikiPath answers the question:  
**“How many clicks does it take to get from one Wikipedia page to another?”**

By following links in the body of Wikipedia articles, it attempts to find a path from a given **source** to a **sink** page.

## Requirements

- Go 1.18+
- Internet connection (the tool fetches live HTML from Wikipedia)

## Usage

### Flags
| Flag           | Description                                         | Default                                    |
| -------------- | --------------------------------------------------- | ------------------------------------------ |
| `-source`      | Starting Wikipedia URL                              | `https://en.wikipedia.org/wiki/Knowledge`  |
| `-sink`        | Target Wikipedia URL                                | `https://en.wikipedia.org/wiki/Philosophy` |
| `-concurrency` | Number of concurrent requests while crawling (1–10) | `3`                                        |



### Build the binary

```bash
git clone https://github.com/michaelginalick/wiki-links.git
cd wikipath/cmd
go build -o main
```
### Run It
./main -source https://en.wikipedia.org/wiki/Dan_Bejar \
       -sink https://en.wikipedia.org/wiki/Philosophy \
       -concurrency 5

### How It Works
WikiPath uses a depth-first search (DFS) strategy to crawl from the source page.
Each thread fetches and parses pages concurrently, speeding up discovery.
Only internal Wikipedia article links are followed—sidebars, references, and other non-content links are ignored.

Note: The core DFS traversal logic is adapted from an example in The Go Programming Language by Alan A. A. Donovan and Brian W. Kernighan.

### Concurrency Notes
Increasing concurrency speeds up crawling, but excessive requests can lead to throttling by Wikipedia.

The tool caps concurrency between 1 and 10 threads for safety.