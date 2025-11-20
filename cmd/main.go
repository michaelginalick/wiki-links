package main

import (
	"flag"
	"fmt"
	"os"
	wikipath "wikipaths/pkg/wikipaths"
)

func main() {
	source := flag.String("source", "https://en.wikipedia.org/wiki/Knowledge", "starting link")
	sink := flag.String("sink", "https://en.wikipedia.org/wiki/Philosophy", "ending link")
	threadCount := flag.Int("concurrency", 3, "number of active threads. Max 10, min 1")

	flag.Parse()
	app, err := wikipath.New(wikipath.WithSourceLink(*source),
		wikipath.WithSinkLink(*sink),
		wikipath.WithThreadCount(*threadCount))
	if err != nil {
		fmt.Println("an error occured %w", err)
		os.Exit(1)
	}
	app.Run()
}
