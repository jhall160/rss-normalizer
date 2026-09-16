package main

import (
	"os"
	"testing"
	"time"
)

func loadFeed(t *testing.T, path string) *Feed {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("opening %s: %v", path, err)
	}
	defer f.Close()

	feed, err := ParseFeed(f)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	return feed
}

// wantParsed checks the wall-clock fields of a parsed date without
// asserting on its zone offset: for zone abbreviations Go doesn't
// recognize (MST in these fixtures isn't the test runner's local
// zone), time.Parse fabricates an offset rather than looking one up,
// but it always preserves the year/month/day/hour/minute exactly as
// written, which is what actually matters here.
func wantParsed(t *testing.T, item Item, year int, month time.Month, day, hour, min int) {
	t.Helper()
	if item.Published == nil {
		t.Fatalf("%s: expected a parsed date, got none (RawDate=%q)", item.Title, item.RawDate)
	}
	got := item.Published
	if got.Year() != year || got.Month() != month || got.Day() != day || got.Hour() != hour || got.Minute() != min {
		t.Errorf("%s: got %v, want %04d-%02d-%02d %02d:%02d", item.Title, got, year, month, day, hour, min)
	}
	if item.RawDate != "" {
		t.Errorf("%s: RawDate should be empty once parsed, got %q", item.Title, item.RawDate)
	}
}

func wantUnparsed(t *testing.T, item Item, raw string) {
	t.Helper()
	if item.Published != nil {
		t.Errorf("%s: expected no parsed date, got %v", item.Title, item.Published)
	}
	if item.RawDate != raw {
		t.Errorf("%s: RawDate = %q, want %q", item.Title, item.RawDate, raw)
	}
}

func TestParseFeedRSSDates(t *testing.T) {
	feed := loadFeed(t, "testdata/rss_dates.xml")
	if len(feed.Items) != 5 {
		t.Fatalf("got %d items, want 5", len(feed.Items))
	}

	wantParsed(t, feed.Items[0], 2006, time.January, 2, 15, 4)
	wantParsed(t, feed.Items[1], 2006, time.January, 2, 15, 4)
	wantParsed(t, feed.Items[2], 2006, time.January, 2, 0, 0)
	wantUnparsed(t, feed.Items[3], "next Tuesday-ish")

	last := feed.Items[4]
	if last.Published != nil || last.RawDate != "" {
		t.Errorf("item with no pubDate: got Published=%v RawDate=%q, want both unset", last.Published, last.RawDate)
	}
}

func TestParseFeedAtomDates(t *testing.T) {
	feed := loadFeed(t, "testdata/atom_dates.xml")
	if len(feed.Items) != 3 {
		t.Fatalf("got %d items, want 3", len(feed.Items))
	}

	wantParsed(t, feed.Items[0], 2021, time.June, 1, 9, 0)
	wantParsed(t, feed.Items[1], 2021, time.June, 3, 9, 0)
	wantUnparsed(t, feed.Items[2], "whenever I got around to it")
}

func TestParseDateMalformed(t *testing.T) {
	cases := []string{
		"",
		"not a date",
		"2021-13-40",
		"Mon, 32 Jan 2021 25:99:99 +0000",
		"Jan 2021",
	}
	for _, c := range cases {
		if _, ok := parseDate(c); ok {
			t.Errorf("parseDate(%q): got ok=true, want false", c)
		}
	}
}
