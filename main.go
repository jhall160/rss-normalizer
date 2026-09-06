package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	jsonOut := flag.Bool("json", false, "emit the normalized feed as JSON instead of human-readable text")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [--json] [file]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Reads an RSS feed (from file, or stdin if no file is given) and prints a\nnormalized version: entities decoded, whitespace collapsed, dates parsed.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	var r io.Reader = os.Stdin
	if flag.NArg() > 0 {
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			fmt.Fprintln(os.Stderr, "rssnorm:", err)
			os.Exit(1)
		}
		defer f.Close()
		r = f
	}

	feed, err := ParseRSS(r)
	if err != nil {
		fmt.Fprintln(os.Stderr, "rssnorm:", err)
		os.Exit(1)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(feed); err != nil {
			fmt.Fprintln(os.Stderr, "rssnorm:", err)
			os.Exit(1)
		}
		return
	}

	WriteText(os.Stdout, feed)
}
