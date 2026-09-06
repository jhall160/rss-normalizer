package main

import (
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"strings"
	"time"
)

// Feed is the normalized shape produced from whatever mess of RSS came in.
type Feed struct {
	Title       string `json:"title"`
	Link        string `json:"link"`
	Description string `json:"description"`
	Items       []Item `json:"items"`
}

type Item struct {
	Title       string     `json:"title"`
	Link        string     `json:"link"`
	Description string     `json:"description"`
	GUID        string     `json:"guid"`
	Published   *time.Time `json:"published,omitempty"`
	// RawDate is set when PubDate was present but didn't match any
	// layout we know about, so callers can still see the original value.
	RawDate string `json:"raw_date,omitempty"`
}

type rawRSS struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Items       []rawItem `xml:"item"`
	} `xml:"channel"`
}

type rawItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	GUID        string `xml:"guid"`
	PubDate     string `xml:"pubDate"`
}

// dateLayouts covers pubDate formats actually seen in feeds in the wild.
// RFC 822 (as amended by RFC 2822) is what the RSS spec asks for, but
// real feeds routinely drop seconds, use non-standard zone abbreviations,
// or just paste in an ISO 8601 timestamp from whatever CMS generated them.
var dateLayouts = []string{
	time.RFC1123Z,
	time.RFC1123,
	time.RFC822Z,
	time.RFC822,
	time.RFC3339,
	"Mon, 2 Jan 2006 15:04:05 -0700",
	"Mon, 2 Jan 2006 15:04:05 MST",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

// ParseRSS reads an RSS 2.0 document and returns its normalized form.
func ParseRSS(r io.Reader) (*Feed, error) {
	dec := xml.NewDecoder(r)
	dec.Strict = false
	// Feeds frequently lie about their own encoding. Rather than fail
	// the whole parse over a charset mismatch, pass bytes through as-is.
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		return input, nil
	}

	var raw rawRSS
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("parsing rss: %w", err)
	}

	feed := &Feed{
		Title:       clean(raw.Channel.Title),
		Link:        clean(raw.Channel.Link),
		Description: clean(raw.Channel.Description),
	}

	for _, ri := range raw.Channel.Items {
		item := Item{
			Title:       clean(ri.Title),
			Link:        clean(ri.Link),
			Description: clean(ri.Description),
			GUID:        clean(ri.GUID),
		}
		if item.GUID == "" {
			item.GUID = item.Link
		}
		if d := clean(ri.PubDate); d != "" {
			if t, ok := parseDate(d); ok {
				item.Published = &t
			} else {
				item.RawDate = d
			}
		}
		feed.Items = append(feed.Items, item)
	}

	return feed, nil
}

// clean turns whatever mangled text a feed handed us - double-escaped
// entities, text wrapped across lines, stray tabs - into one trimmed
// line of plain text.
func clean(s string) string {
	s = html.UnescapeString(s)
	return strings.Join(strings.Fields(s), " ")
}

func parseDate(s string) (time.Time, bool) {
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
