package main

import (
	"fmt"
	"io"
)

// WriteText renders a Feed for a human reading a terminal. JSON output
// (see main.go's --json flag) covers the machine-reading case.
func WriteText(w io.Writer, feed *Feed) {
	fmt.Fprintf(w, "%s\n", feed.Title)
	if feed.Link != "" {
		fmt.Fprintf(w, "%s\n", feed.Link)
	}
	if feed.Description != "" {
		fmt.Fprintf(w, "\n%s\n", feed.Description)
	}
	fmt.Fprintf(w, "\n%d items\n", len(feed.Items))

	for i, item := range feed.Items {
		fmt.Fprintf(w, "\n%d. %s\n", i+1, item.Title)
		if item.Link != "" {
			fmt.Fprintf(w, "   %s\n", item.Link)
		}
		switch {
		case item.Published != nil:
			fmt.Fprintf(w, "   %s\n", item.Published.Format("2006-01-02 15:04"))
		case item.RawDate != "":
			fmt.Fprintf(w, "   %s (unrecognized date format)\n", item.RawDate)
		}
		if item.Description != "" {
			fmt.Fprintf(w, "   %s\n", truncate(item.Description, 200))
		}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
