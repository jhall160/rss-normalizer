package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"strings"
	"time"
)

// Feed is the normalized shape produced from whatever mess of RSS or
// Atom came in.
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
	// Content holds the full item body when the feed provides one
	// separately from its (often truncated) description: RSS's
	// content:encoded extension, or Atom's <content> element.
	Content string `json:"content,omitempty"`
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
	// Encoded is the content:encoded element from the RSS content
	// module (http://purl.org/rss/1.0/modules/content/), used by
	// most feed generators (WordPress included) to carry the full
	// HTML body alongside a shorter plain-text description.
	Encoded string `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
}

// rawAtom mirrors the parts of an Atom 1.0 <feed> we care about. Atom
// links are elements with an href attribute rather than element text,
// and there can be several of them (alternate, self, edit, ...), so
// they need their own type instead of a plain string field.
type rawAtom struct {
	Title    string     `xml:"title"`
	Subtitle string     `xml:"subtitle"`
	Links    []atomLink `xml:"link"`
	Entries  []rawEntry `xml:"entry"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

type rawEntry struct {
	Title     string     `xml:"title"`
	Links     []atomLink `xml:"link"`
	ID        string     `xml:"id"`
	Published string     `xml:"published"`
	Updated   string     `xml:"updated"`
	Summary   string     `xml:"summary"`
	Content   string     `xml:"content"`
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
	time.RFC3339Nano,
	"Mon, 2 Jan 2006 15:04:05 -0700",
	"Mon, 2 Jan 2006 15:04:05 MST",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

// ParseFeed reads an RSS 2.0 or Atom 1.0 document and returns its
// normalized form, picking the right parser based on the root element.
func ParseFeed(r io.Reader) (*Feed, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading feed: %w", err)
	}

	root, err := rootElement(data)
	if err != nil {
		return nil, fmt.Errorf("parsing feed: %w", err)
	}

	if root == "feed" {
		return parseAtom(data)
	}
	return parseRSS(data)
}

// newDecoder sets up an xml.Decoder the way every parser here needs it:
// lenient about malformed markup, and indifferent to bogus encoding
// declarations, since feeds frequently lie about their own charset.
func newDecoder(data []byte) *xml.Decoder {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		return input, nil
	}
	return dec
}

// rootElement returns the local name of the document's root element,
// which is enough to tell an RSS <rss> from an Atom <feed>.
func rootElement(data []byte) (string, error) {
	dec := newDecoder(data)
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", err
		}
		if se, ok := tok.(xml.StartElement); ok {
			return se.Name.Local, nil
		}
	}
}

func parseRSS(data []byte) (*Feed, error) {
	var raw rawRSS
	if err := newDecoder(data).Decode(&raw); err != nil {
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
			Content:     clean(ri.Encoded),
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

func parseAtom(data []byte) (*Feed, error) {
	var raw rawAtom
	if err := newDecoder(data).Decode(&raw); err != nil {
		return nil, fmt.Errorf("parsing atom: %w", err)
	}

	feed := &Feed{
		Title:       clean(raw.Title),
		Link:        clean(atomHref(raw.Links)),
		Description: clean(raw.Subtitle),
	}

	for _, re := range raw.Entries {
		item := Item{
			Title:       clean(re.Title),
			Link:        clean(atomHref(re.Links)),
			Description: clean(firstNonEmpty(re.Summary, re.Content)),
			GUID:        clean(re.ID),
			Content:     clean(re.Content),
		}
		if item.GUID == "" {
			item.GUID = item.Link
		}
		// Atom entries carry <published> (when the entry was first
		// created) and <updated> (last change), and only the latter
		// is required by the spec. Fall back to it when there's no
		// published date, same as a feed reader would.
		if d := clean(firstNonEmpty(re.Published, re.Updated)); d != "" {
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

// atomHref picks the link to surface for a feed or entry that may have
// several: prefer the one marked rel="alternate" (the human-readable
// page), falling back to a relless link, then whatever comes first.
func atomHref(links []atomLink) string {
	var relless, first string
	for _, l := range links {
		if first == "" {
			first = l.Href
		}
		switch l.Rel {
		case "alternate":
			return l.Href
		case "":
			if relless == "" {
				relless = l.Href
			}
		}
	}
	if relless != "" {
		return relless
	}
	return first
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
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
