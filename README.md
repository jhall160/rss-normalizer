# rssnorm

RSS feeds in the wild are inconsistent in ways that don't show up until
something downstream chokes on them: titles with double-escaped HTML
entities (`&amp;amp;` instead of `&amp;`), descriptions wrapped across
multiple lines with stray tabs, `pubDate` values that ignore RFC 822 and
paste in whatever format the CMS felt like that day, missing `guid`
elements, and so on.

`rssnorm` reads an RSS 2.0 or Atom 1.0 document and prints a normalized
version of it: entities decoded once, whitespace collapsed to single
spaces, dates parsed where possible (and passed through as-is, flagged,
when they aren't), and a `guid` filled in from the item link when one is
missing. The format is detected from the document's root element, so
there's nothing to tell it which one you're feeding it.

## Usage

```sh
go run . feed.xml
```

Or read from stdin:

```sh
curl -s https://example.com/feed.xml | go run .
```

Human-readable output looks like:

```
Example Blog
https://example.com

Assorted posts about nothing in particular

2 items

1. Weekend reading &amp; links
   https://example.com/posts/42
   2026-08-30 09:15
   A short roundup of things worth reading this week.

2. Untitled draft
   https://example.com/posts/41
   Thu 28 Aug 2026 (unrecognized date format)
```

Pass `--json` to get the same normalized data as JSON instead, for
piping into something else:

```sh
go run . --json feed.xml
```

```json
{
  "title": "Example Blog",
  "link": "https://example.com",
  "description": "Assorted posts about nothing in particular",
  "items": [
    {
      "title": "Weekend reading & links",
      "link": "https://example.com/posts/42",
      "description": "A short roundup of things worth reading this week.",
      "guid": "https://example.com/posts/42",
      "published": "2026-08-30T09:15:00Z"
    },
    {
      "title": "Untitled draft",
      "link": "https://example.com/posts/41",
      "description": "",
      "guid": "https://example.com/posts/41",
      "raw_date": "Thu 28 Aug 2026"
    }
  ]
}
```

## Status

Handles RSS 2.0 and Atom 1.0. Only standard library, no dependencies.
RSS's `content:encoded` extension and Atom's `<content>` element are
picked up into a separate `content` field alongside `description`, since
feeds often use the latter for a short summary and the former for the
full body. Other namespaced extensions (`dc:creator` and friends) aren't
read yet.

## License

MIT, see LICENSE.
