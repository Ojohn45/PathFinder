package main

import (
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
)

// everyTopic returns every topic on the site exactly once: the ones in the
// otherTopics map, plus the programming languages built at start-up.
func everyTopic() []Topic {
	seen := map[string]bool{}
	var out []Topic

	add := func(t Topic) {
		if seen[t.Slug] {
			return
		}
		seen[t.Slug] = true
		out = append(out, t)
	}

	for _, t := range otherTopics {
		add(t)
	}
	for _, t := range programmingTopicList() {
		add(t)
	}
	for _, t := range topicBySlug {
		add(t)
	}
	return out
}

// 1. Every topic has exactly fifteen places.
func TestEveryTopicHasEnoughResources(t *testing.T) {
	for _, topic := range everyTopic() {
		want := topic.Want
		if want == 0 {
			want = 15
		}
		// Want is now a floor, not a cap: a topic may hold as many verified
		// places as we can find, but never fewer than it promises.
		if got := len(topic.Resources); got < want {
			t.Errorf("topic %q has %d places, want at least %d", topic.Slug, got, want)
		}
	}
}

// 2. Every resource ID is unique across the whole site.
func TestResourceIDsAreUnique(t *testing.T) {
	seen := map[string]string{}
	for _, topic := range everyTopic() {
		for _, r := range topic.Resources {
			if r.ID == "" {
				t.Errorf("topic %q has a place with no ID", topic.Slug)
				continue
			}
			if where, ok := seen[r.ID]; ok {
				t.Errorf("duplicate ID %q in %q and %q", r.ID, where, topic.Slug)
				continue
			}
			seen[r.ID] = topic.Slug
		}
	}
}

// 3. Every URL is https and parses; 4. no duplicate URL inside one topic.
func TestResourceURLs(t *testing.T) {
	for _, topic := range everyTopic() {
		seen := map[string]bool{}
		for _, r := range topic.Resources {
			if !strings.HasPrefix(r.URL, "https://") {
				t.Errorf("%s: %q is not https: %s", topic.Slug, r.ID, r.URL)
				continue
			}
			u, err := url.Parse(r.URL)
			if err != nil {
				t.Errorf("%s: %q does not parse: %v", topic.Slug, r.ID, err)
				continue
			}
			if u.Host == "" {
				t.Errorf("%s: %q has no host: %s", topic.Slug, r.ID, r.URL)
			}
			if seen[r.URL] {
				t.Errorf("%s: %q repeats the URL %s", topic.Slug, r.ID, r.URL)
			}
			seen[r.URL] = true
		}
	}
}

// 5. Every section slug and theme key is unique.
func TestSectionSlugsAndThemesAreUnique(t *testing.T) {
	slugs := map[string]bool{}
	themes := map[string]string{}
	for _, s := range sections {
		if slugs[s.Slug] {
			t.Errorf("duplicate section slug %q", s.Slug)
		}
		slugs[s.Slug] = true

		if s.Theme == "" {
			t.Errorf("section %q has no theme key", s.Slug)
			continue
		}
		if where, ok := themes[s.Theme]; ok {
			t.Errorf("theme %q used by both %q and %q", s.Theme, where, s.Slug)
			continue
		}
		themes[s.Theme] = s.Slug
	}
}

// 6. Every theme key has a matching block in static/style.css.
func TestEveryThemeHasCSS(t *testing.T) {
	css, err := os.ReadFile("static/style.css")
	if err != nil {
		t.Fatalf("could not read static/style.css: %v", err)
	}
	sheet := string(css)

	for _, s := range sections {
		block := regexp.MustCompile(`\[data-theme="` + regexp.QuoteMeta(s.Theme) + `"\]`)
		if !block.MatchString(sheet) {
			t.Errorf("section %q (theme %q) has no [data-theme] block", s.Slug, s.Theme)
		}
		card := regexp.MustCompile(`\.sect-card\.theme-` + regexp.QuoteMeta(s.Theme) + `\s*\{`)
		if !card.MatchString(sheet) {
			t.Errorf("section %q (theme %q) has no .sect-card.theme-%s line", s.Slug, s.Theme, s.Theme)
		}
	}
}
