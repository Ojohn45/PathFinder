package main

import (
	"net/http"
	"time"
)

// ---------- History ----------
//
// Every time a visitor opens one of the fifteen places, the visit is written to
// storage/history.json. This page shows the whole trail, newest first, grouped
// by day, with a link to go straight back to that place.

// HistoryRow is one line on the page: the visit plus the names it belongs to.
type HistoryRow struct {
	ResourceName string
	LanguageName string // the topic: a language, "Anime", "Read Manga"...
	SectionName  string
	SectionSlug  string
	Link         string // /go/<topic>/<resource>
	When         time.Time
}

// HistoryDay groups the visits of one calendar day.
type HistoryDay struct {
	Heading string
	Rows    []HistoryRow
}

// allHistory returns every visit for one user, newest first.
func allHistory(userID int) []HistoryEntry {
	mu.Lock()
	defer mu.Unlock()
	var out []HistoryEntry
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].UserID == userID {
			out = append(out, history[i])
		}
	}
	return out
}

// historyRows turns raw visits into rows with human names attached.
func historyRows(entries []HistoryEntry) []HistoryRow {
	rows := make([]HistoryRow, 0, len(entries))
	for _, e := range entries {
		row := HistoryRow{
			ResourceName: e.ResourceName,
			When:         e.ClickedAt,
			Link:         "/go/" + e.LanguageSlug + "/" + e.ResourceID,
			LanguageName: e.LanguageSlug,
			SectionName:  "PathFinder",
		}
		if t, ok := topicBySlug[e.LanguageSlug]; ok {
			row.LanguageName = t.Name
		}
		if slug, ok := sectionOfTopic[e.LanguageSlug]; ok {
			if sec, ok := sectionBySlug[slug]; ok {
				row.SectionName = sec.Name
				row.SectionSlug = sec.Slug
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// groupByDay puts rows under Today / Yesterday / a date heading.
func groupByDay(rows []HistoryRow) []HistoryDay {
	var days []HistoryDay
	now := time.Now()
	for _, row := range rows {
		when := row.When.Local()
		heading := when.Format("Mon 2 Jan 2006")
		switch {
		case sameDay(when, now):
			heading = "Today"
		case sameDay(when, now.AddDate(0, 0, -1)):
			heading = "Yesterday"
		}
		if len(days) == 0 || days[len(days)-1].Heading != heading {
			days = append(days, HistoryDay{Heading: heading})
		}
		days[len(days)-1].Rows = append(days[len(days)-1].Rows, row)
	}
	return days
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// historyPageHandler serves /history.
func historyPageHandler(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if u == nil {
		guestRedirect(w, r)
		return
	}
	entries := allHistory(u.ID)
	rows := historyRows(entries)
	data := PageData{
		User:      u,
		Theme:     "base",
		TotalSeen: len(entries),
		History:   groupByDay(rows),
	}
	render(w, "history.html", data)
}
