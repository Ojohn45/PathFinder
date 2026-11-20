package main

import (
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// ---------- Searching INSIDE a section ----------
//
// There is no search bar on the landing page, the home page or Browse: you have
// to be inside a section first. Only five sections carry one:
//
//	Programming     search the languages themselves
//	Entertainment   search a title, then see where to watch, listen or play it
//	Read Manga      search a title, then see where to read it
//	Read Manhwa     same
//	Read Comix      same
//
// Learning, Lifestyle and Create & Work deliberately have none.

// SearchLink is one place you can go for a searched title.
type SearchLink struct {
	Name        string
	URL         string
	Description string
}

// SearchGroup is a row of places, e.g. "Watch it" or "Listen to it".
type SearchGroup struct {
	Title string
	Blurb string
	Links []SearchLink
}

// SectionSearch is the whole answer to "where can I find this?".
type SectionSearch struct {
	Placeholder string
	Blurb       string
	Groups      []SearchGroup
}

// searchable says which sections get a search bar.
func searchable(slug string) bool {
	switch slug {
	case "programming", "entertainment", "manga", "manhwa", "comix":
		return true
	}
	return false
}

func link(name, href, desc string) SearchLink {
	return SearchLink{Name: name, URL: href, Description: desc}
}

// watchLinks answers "where can I watch this film, anime or cartoon?".
func watchLinks(q string) []SearchLink {
	e := url.QueryEscape(q)
	return []SearchLink{
		link("JustWatch", "https://www.justwatch.com/us/search?q="+e, "Every streaming service at once, tells you who has it"),
		link("Netflix", "https://www.netflix.com/search?q="+e, "Search Netflix directly"),
		link("Prime Video", "https://www.amazon.com/s?k="+e+"&i=instant-video", "Amazon's film and series catalogue"),
		link("Apple TV", "https://tv.apple.com/search?term="+e, "Apple's store, rent or buy"),
		link("Crunchyroll", "https://www.crunchyroll.com/search?q="+e, "The big anime library"),
		link("IMDb", "https://www.imdb.com/find/?q="+e, "Find the exact title, year and cast first"),
		link("YouTube", "https://www.youtube.com/results?search_query="+e, "Trailers, clips and full uploads"),
		link("MyAnimeList", "https://myanimelist.net/search/all?q="+e, "Anime and manga, with ratings"),
		link("AniList", "https://anilist.co/search/anime?search="+e, "Anime search with streaming links"),
	}
}

// listenLinks answers "where can I listen to this?".
func listenLinks(q string) []SearchLink {
	e := url.QueryEscape(q)
	return []SearchLink{
		link("Spotify", "https://open.spotify.com/search/"+e, "Songs, albums, artists and podcasts"),
		link("YouTube Music", "https://music.youtube.com/search?q="+e, "Free with ads, huge catalogue"),
		link("Apple Music", "https://music.apple.com/us/search?term="+e, "Apple's catalogue"),
		link("Audiomack", "https://audiomack.com/search?q="+e, "Strong on African music, free streaming"),
		link("SoundCloud", "https://soundcloud.com/search?q="+e, "Independent artists and mixes"),
		link("Deezer", "https://www.deezer.com/search/"+e, "Streaming with a free tier"),
		link("Bandcamp", "https://bandcamp.com/search?q="+e, "Buy directly from the artist"),
	}
}

// playLinks answers "where can I play this game?".
func playLinks(q string) []SearchLink {
	e := url.QueryEscape(q)
	return []SearchLink{
		link("Steam", "https://store.steampowered.com/search/?term="+e, "PC games, the biggest store"),
		link("Epic Games Store", "https://store.epicgames.com/en-US/browse?q="+e, "Weekly free games too"),
		link("GOG", "https://www.gog.com/en/games?query="+e, "DRM-free PC games, old and new"),
		link("itch.io", "https://itch.io/search?q="+e, "Indie games, many free"),
		link("PlayStation Store", "https://store.playstation.com/en-us/search/"+e, "PS4 and PS5 games"),
		link("Xbox Store", "https://www.xbox.com/en-US/search?q="+e, "Xbox and PC games"),
		link("Nintendo eShop", "https://www.nintendo.com/us/search/?q="+e, "Switch games"),
		link("Google Play", "https://play.google.com/store/search?q="+e, "Android games"),
	}
}

// mangaReadLinks answers "where can I read this manga?".
func mangaReadLinks(q string) []SearchLink {
	e := url.QueryEscape(q)
	return []SearchLink{
		link("MangaDex", "https://mangadex.org/search?title="+e, "The biggest free library, fan-run"),
		link("MANGA Plus by Shueisha", "https://mangaplus.shueisha.co.jp/search?query="+e, "Official and free, same-day Shonen Jump"),
		link("VIZ Media", "https://www.viz.com/search?search="+e, "Official English manga"),
		link("MyAnimeList", "https://myanimelist.net/search/all?q="+e, "Find the exact series and its rating"),
		link("AniList", "https://anilist.co/search/manga?search="+e, "Manga search with stats"),
		link("MangaUpdates", "https://www.mangaupdates.com/series.html?search="+e, "Track releases and scanlation status"),
		link("Crunchyroll Manga", "https://www.crunchyroll.com/search?q="+e, "Manga included with a Crunchyroll account"),
	}
}

// manhwaReadLinks answers "where can I read this manhwa?".
func manhwaReadLinks(q string) []SearchLink {
	e := url.QueryEscape(q)
	return []SearchLink{
		link("WEBTOON", "https://www.webtoons.com/en/search?keyword="+e, "The home of Korean webtoons, free chapters"),
		link("Tapas", "https://tapas.io/search?q="+e, "Official and indie, plenty free"),
		link("Tappytoon", "https://www.tappytoon.com/en/search?q="+e, "Official manhwa, romance and action"),
		link("Lezhin Comics", "https://www.lezhin.com/en/search?q="+e, "Official manhwa, mature titles"),
		link("Manta", "https://manta.net/en/search?q="+e, "Unlimited official manhwa subscription"),
		link("MangaDex", "https://mangadex.org/search?title="+e, "Big free library, manhwa included"),
		link("MangaUpdates", "https://www.mangaupdates.com/series.html?search="+e, "Track what you are reading"),
	}
}

// comixReadLinks answers "where can I read this comic?".
func comixReadLinks(q string) []SearchLink {
	e := url.QueryEscape(q)
	return []SearchLink{
		link("GlobalComix", "https://globalcomix.com/search?query="+e, "Huge indie and publisher catalogue"),
		link("Kindle & comiXology", "https://www.amazon.com/s?k="+e+"&i=digital-text", "Buy digital comics for any device"),
		link("GoComics", "https://www.gocomics.com/search?q="+e, "Daily newspaper strips"),
		link("League of Comic Geeks", "https://leagueofcomicgeeks.com/search?keyword="+e, "Track series and release dates"),
		link("MyAnimeList", "https://myanimelist.net/search/all?q="+e, "Also lists many western comics"),
		link("Amazon Books", "https://www.amazon.com/s?k="+e+"+comic&i=stripbooks", "Graphic novels in print and digital"),
	}
}

// searchFor builds the whole answer for one section and one query.
func searchFor(slug, q string) (SectionSearch, bool) {
	switch slug {
	case "entertainment":
		return SectionSearch{
			Placeholder: "Type a film, anime, cartoon, song or game name",
			Blurb:       "Entertainment searches titles, not websites: type what you want and PathFinder points at every place that has it.",
			Groups: []SearchGroup{
				{Title: "Watch it", Blurb: "Films, series, anime and cartoons", Links: watchLinks(q)},
				{Title: "Listen to it", Blurb: "Songs, albums, artists and mixes", Links: listenLinks(q)},
				{Title: "Play it", Blurb: "Games on PC, console and phone", Links: playLinks(q)},
			},
		}, true
	case "manga":
		return SectionSearch{
			Placeholder: "Type a manga title",
			Blurb:       "Type the name of a manga and PathFinder shows where to read it, official publishers first.",
			Groups:      []SearchGroup{{Title: "Read it", Blurb: "Official and free libraries", Links: mangaReadLinks(q)}},
		}, true
	case "manhwa":
		return SectionSearch{
			Placeholder: "Type a manhwa title",
			Blurb:       "Type the name of a manhwa and PathFinder shows where to read it, official platforms first.",
			Groups:      []SearchGroup{{Title: "Read it", Blurb: "Official and free platforms", Links: manhwaReadLinks(q)}},
		}, true
	case "comix":
		return SectionSearch{
			Placeholder: "Type a comic title",
			Blurb:       "Type the name of a comic and PathFinder shows where to read it.",
			Groups:      []SearchGroup{{Title: "Read it", Blurb: "Official stores and free libraries", Links: comixReadLinks(q)}},
		}, true
	case "programming":
		return SectionSearch{
			Placeholder: "Type a language name (Rust, COBOL, Gleam…)",
			Blurb:       "Programming only searches languages, because that is what this section holds.",
		}, true
	}
	return SectionSearch{}, false
}

// searchLanguages finds languages by name. "js" and "c#" style nicknames work.
func searchLanguages(q string) []Topic {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return nil
	}
	alias := map[string]string{
		"js": "javascript", "ts": "typescript", "golang": "go", "py": "python",
		"c++": "cpp", "c plus plus": "cpp", "c#": "csharp", "f#": "fsharp",
		"vb": "vbnet", "vb.net": "vbnet", "postgres": "sql", "mysql": "sql",
	}
	if slug, ok := alias[q]; ok {
		if t, ok := programmingTopics[slug]; ok {
			return []Topic{t}
		}
	}
	var out []Topic
	for _, t := range programmingTopics {
		if strings.Contains(strings.ToLower(t.Name), q) || strings.Contains(t.Slug, q) {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// searchHandlerFor serves /section/<slug>/search?q=... . Each searchable section
// gets its own route, because a {slug} wildcard would clash with the
// programming tab routes (/section/programming/docs and friends).
func searchHandlerFor(slug string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sectionSearch(w, r, slug)
	}
}

func sectionSearch(w http.ResponseWriter, r *http.Request, slug string) {
	u := currentUser(r)
	if u == nil {
		guestRedirect(w, r)
		return
	}
	sec, ok := sectionBySlug[slug]
	if !ok || !searchable(slug) {
		http.NotFound(w, r)
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	data := PageData{User: u, Section: sec, Theme: sec.Theme, Query: query}
	data.Search, _ = searchFor(slug, query)
	data.Count = len(query)

	if slug == "programming" && query != "" {
		data.Languages = searchLanguages(query)
	}

	render(w, "search.html", data)
}
