package main

import (
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// The avatar system: one picture per account, used everywhere the account
// appears. Uploaded pictures are re-encoded and shrunk on the server, then kept
// in Postgres next to the account, so they survive a redeploy.

const (
	avatarSize    = 256     // saved as a 256 x 256 square
	avatarMaxBody = 6 << 20 // whole upload request
	avatarMaxFile = 4 << 20 // the picture itself
)

// ---------- reading and writing the account ----------

func userByID(id int) (User, bool) {
	mu.Lock()
	defer mu.Unlock()
	for _, u := range users {
		if u.ID == id {
			return u, true
		}
	}
	return User{}, false
}

// avatarPath is what the <img> tag asks for. The ?v= means a new picture is
// never hidden behind an old cached one.
func avatarPath(u *User) string {
	if u == nil || u.AvatarKind == "" {
		return ""
	}
	return fmt.Sprintf("/avatar/%d?v=%d", u.ID, u.AvatarAt.Unix())
}

func saveAvatar(id int, kind, ref string, data []byte) error {
	mu.Lock()
	defer mu.Unlock()
	for i := range users {
		if users[i].ID != id {
			continue
		}
		oldKind, oldRef, oldData, oldAt := users[i].AvatarKind, users[i].AvatarRef, users[i].AvatarData, users[i].AvatarAt
		users[i].AvatarKind, users[i].AvatarRef, users[i].AvatarData = kind, ref, data
		users[i].AvatarAt = time.Now()
		if err := saveJSON(usersFile, users); err != nil {
			users[i].AvatarKind, users[i].AvatarRef, users[i].AvatarData, users[i].AvatarAt = oldKind, oldRef, oldData, oldAt
			return err
		}
		return nil
	}
	return errors.New("no such user")
}

func saveProfileText(id int, displayName, bio string) error {
	mu.Lock()
	defer mu.Unlock()
	for i := range users {
		if users[i].ID != id {
			continue
		}
		oldName, oldBio := users[i].DisplayName, users[i].Bio
		users[i].DisplayName, users[i].Bio = displayName, bio
		if err := saveJSON(usersFile, users); err != nil {
			users[i].DisplayName, users[i].Bio = oldName, oldBio
			return err
		}
		return nil
	}
	return errors.New("no such user")
}

func saveProfileStyle(id int, frame, banner string, calm bool) error {
	mu.Lock()
	defer mu.Unlock()
	for i := range users {
		if users[i].ID != id {
			continue
		}
		oldF, oldB, oldC := users[i].Frame, users[i].Banner, users[i].CalmMode
		users[i].Frame, users[i].Banner, users[i].CalmMode = frame, banner, calm
		if err := saveJSON(usersFile, users); err != nil {
			users[i].Frame, users[i].Banner, users[i].CalmMode = oldF, oldB, oldC
			return err
		}
		return nil
	}
	return errors.New("no such user")
}

// ---------- serving the picture ----------

func avatarServeHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u, ok := userByID(id)
	if !ok || u.AvatarKind == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=604800")
	if u.AvatarKind == "upload" && len(u.AvatarData) > 0 {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("ETag", fmt.Sprintf(`"u%d-%d"`, u.ID, u.AvatarAt.Unix()))
		w.Write(u.AvatarData)
		return
	}
	svg, ok := builtinAvatarSVG(u.AvatarRef)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("ETag", fmt.Sprintf(`"b%d-%s"`, u.ID, u.AvatarRef))
	w.Write(svg)
}

// ---------- turning an upload into a small square picture ----------

func shrinkToAvatar(r io.Reader) ([]byte, error) {
	src, _, err := image.Decode(io.LimitReader(r, avatarMaxFile))
	if err != nil {
		return nil, errors.New("that file is not a picture we can read. Try a JPG, PNG or WebP image")
	}
	b := src.Bounds()
	side := b.Dx()
	if b.Dy() < side {
		side = b.Dy()
	}
	if side <= 0 {
		return nil, errors.New("that picture looks empty")
	}
	// take the middle square, so nothing is stretched
	offX := b.Min.X + (b.Dx()-side)/2
	offY := b.Min.Y + (b.Dy()-side)/2
	crop := image.Rect(offX, offY, offX+side, offY+side)

	dst := image.NewRGBA(image.Rect(0, 0, avatarSize, avatarSize))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, crop, xdraw.Src, nil)

	return encodeJPEG(dst)
}

func encodeJPEG(img image.Image) ([]byte, error) {
	var b bytesBuffer
	if err := jpeg.Encode(&b, img, &jpeg.Options{Quality: 86}); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// bytesBuffer is a tiny io.Writer that collects bytes.
type bytesBuffer struct{ b []byte }

func (w *bytesBuffer) Write(p []byte) (int, error) { w.b = append(w.b, p...); return len(p), nil }
func (w *bytesBuffer) Bytes() []byte               { return w.b }

// ---------- the built-in gallery (our own artwork) ----------

type AvatarGroup struct {
	Slug  string
	Name  string
	Items []string
}

const avatarsPerCategory = 6

var avatarCategories = []struct{ Slug, Name string }{
	{"anime", "Anime-inspired"},
	{"gaming", "Gaming"},
	{"fantasy", "Fantasy"},
	{"minimal", "Minimalist"},
	{"cartoon", "Cartoon"},
	{"abstract", "Abstract"},
	{"cute", "Cute"},
	{"animal", "Animal"},
	{"future", "Futuristic"},
	{"nature", "Nature"},
}

func avatarGallery() []AvatarGroup {
	groups := make([]AvatarGroup, 0, len(avatarCategories))
	for _, c := range avatarCategories {
		g := AvatarGroup{Slug: c.Slug, Name: c.Name}
		for i := 1; i <= avatarsPerCategory; i++ {
			g.Items = append(g.Items, fmt.Sprintf("%s-%d", c.Slug, i))
		}
		groups = append(groups, g)
	}
	return groups
}

// avatarPalettes gives each category its own colours, all sitting comfortably
// on PathFinder's dark blue background.
var avatarPalettes = map[string][][3]string{
	"anime":    {{"#1b2a5e", "#ff8fb8", "#ffd6e7"}, {"#12224a", "#7cc4ff", "#d8ecff"}},
	"gaming":   {{"#161a3a", "#5ef2a8", "#0d3b2a"}, {"#1a1430", "#a855f7", "#2a1550"}},
	"fantasy":  {{"#20123f", "#c084fc", "#f5d0fe"}, {"#0f1b3d", "#38bdf8", "#c7e9ff"}},
	"minimal":  {{"#141a33", "#94a3b8", "#e2e8f0"}, {"#101828", "#38bdf8", "#e0f2fe"}},
	"cartoon":  {{"#22162e", "#fbbf24", "#7c2d12"}, {"#122033", "#f472b6", "#831843"}},
	"abstract": {{"#101a3a", "#818cf8", "#312e81"}, {"#0c2030", "#22d3ee", "#0e7490"}},
	"cute":     {{"#2a1230", "#fb7185", "#fecdd3"}, {"#241a3a", "#a78bfa", "#ede9fe"}},
	"animal":   {{"#1a2318", "#a3e635", "#365314"}, {"#221a12", "#f59e0b", "#78350f"}},
	"future":   {{"#0a1330", "#22d3ee", "#1e3a8a"}, {"#120a2e", "#c026d3", "#4a044e"}},
	"nature":   {{"#0f2418", "#4ade80", "#14532d"}, {"#0d1f2d", "#2dd4bf", "#134e4a"}},
}

func builtinAvatarSVG(id string) ([]byte, bool) {
	parts := strings.SplitN(id, "-", 2)
	if len(parts) != 2 {
		return nil, false
	}
	cat := parts[0]
	n, err := strconv.Atoi(parts[1])
	if err != nil || n < 1 || n > avatarsPerCategory {
		return nil, false
	}
	pal, ok := avatarPalettes[cat]
	if !ok {
		return nil, false
	}
	p := pal[(n-1)%len(pal)]
	bg, accent, soft := p[0], p[1], p[2]

	motif := avatarMotif(cat, n, accent, soft)
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" width="256" height="256" role="img" aria-label="PathFinder avatar">
<defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="%s"/><stop offset="1" stop-color="#070c22"/></linearGradient></defs>
<rect width="100" height="100" rx="50" fill="url(#g)"/>%s</svg>`, bg, motif)
	return []byte(svg), true
}

func avatarMotif(cat string, n int, accent, soft string) string {
	rot := float64((n - 1) * 12)
	switch cat {
	case "anime":
		eye := 3.2 + float64(n%3)*0.6
		return fmt.Sprintf(`<path d="M50 16c19 0 30 13 30 30v34H20V46c0-17 11-30 30-30z" fill="%s" opacity="0.35"/>
<circle cx="50" cy="52" r="22" fill="#0d1430" opacity="0.55"/>
<circle cx="42" cy="50" r="%.1f" fill="%s"/><circle cx="58" cy="50" r="%.1f" fill="%s"/>
<circle cx="43" cy="49" r="1.4" fill="#0b1020"/><circle cx="59" cy="49" r="1.4" fill="#0b1020"/>
<path d="M44 62q6 4 12 0" stroke="%s" stroke-width="2" fill="none" stroke-linecap="round"/>`, soft, eye, accent, eye, accent, accent)
	case "gaming":
		return fmt.Sprintf(`<g transform="rotate(%.0f 50 50)"><rect x="24" y="38" width="52" height="26" rx="13" fill="%s"/>
<rect x="31" y="47" width="9" height="3" rx="1.5" fill="#0b1020"/><rect x="34" y="44" width="3" height="9" rx="1.5" fill="#0b1020"/>
<circle cx="64" cy="48" r="2.4" fill="#0b1020"/><circle cx="70" cy="54" r="2.4" fill="#0b1020"/>
<circle cx="50" cy="52" r="2" fill="%s"/></g>`, rot, accent, soft)
	case "fantasy":
		return fmt.Sprintf(`<g transform="rotate(%.0f 50 50)"><path d="M50 18l7 20 21 3-15 15 4 21-17-10-17 10 4-21-15-15 21-3z" fill="%s" opacity="0.9"/>
<circle cx="50" cy="50" r="30" fill="none" stroke="%s" stroke-width="1.6" opacity="0.55"/></g>`, rot, accent, soft)
	case "minimal":
		return fmt.Sprintf(`<circle cx="50" cy="50" r="26" fill="none" stroke="%s" stroke-width="4"/>
<path d="M32 58q18 %.0f 36 0" stroke="%s" stroke-width="3" fill="none" stroke-linecap="round"/>`, accent, 10+float64(n), soft)
	case "cartoon":
		return fmt.Sprintf(`<circle cx="50" cy="48" r="24" fill="%s"/>
<circle cx="42" cy="44" r="4" fill="#0b1020"/><circle cx="58" cy="44" r="4" fill="#0b1020"/>
<path d="M38 58q12 12 24 0" stroke="#0b1020" stroke-width="3" fill="none" stroke-linecap="round"/>
<circle cx="30" cy="34" r="5" fill="%s" opacity="0.7"/>`, accent, soft)
	case "abstract":
		return fmt.Sprintf(`<g transform="rotate(%.0f 50 50)" opacity="0.9"><circle cx="38" cy="42" r="20" fill="%s" opacity="0.7"/>
<circle cx="62" cy="48" r="17" fill="%s" opacity="0.65"/><circle cx="48" cy="64" r="14" fill="%s" opacity="0.55"/></g>`, rot, accent, soft, accent)
	case "cute":
		return fmt.Sprintf(`<circle cx="50" cy="50" r="23" fill="%s"/>
<circle cx="43" cy="47" r="2.6" fill="#0b1020"/><circle cx="57" cy="47" r="2.6" fill="#0b1020"/>
<path d="M46 55q4 4 8 0" stroke="#0b1020" stroke-width="2.2" fill="none" stroke-linecap="round"/>
<circle cx="36" cy="55" r="4" fill="%s" opacity="0.6"/><circle cx="64" cy="55" r="4" fill="%s" opacity="0.6"/>`, accent, soft, soft)
	case "animal":
		return fmt.Sprintf(`<path d="M30 34l6-14 10 10z" fill="%s"/><path d="M70 34l-6-14-10 10z" fill="%s"/>
<circle cx="50" cy="52" r="22" fill="%s"/>
<circle cx="43" cy="49" r="2.6" fill="#0b1020"/><circle cx="57" cy="49" r="2.6" fill="#0b1020"/>
<ellipse cx="50" cy="60" rx="7" ry="5" fill="%s"/><circle cx="50" cy="58" r="2.2" fill="#0b1020"/>`, accent, accent, accent, soft)
	case "future":
		return fmt.Sprintf(`<g transform="rotate(%.0f 50 50)"><rect x="26" y="40" width="48" height="20" rx="10" fill="%s" opacity="0.9"/>
<rect x="32" y="46" width="36" height="8" rx="4" fill="%s" opacity="0.85"/>
<path d="M20 68h60M20 74h60" stroke="%s" stroke-width="1.4" opacity="0.5"/></g>`, rot, accent, soft, accent)
	default: // nature
		return fmt.Sprintf(`<circle cx="66" cy="34" r="10" fill="%s" opacity="0.85"/>
<path d="M18 72l18-26 12 16 8-10 20 20z" fill="%s"/>
<path d="M18 78q16 -6 32 0t32 0" stroke="%s" stroke-width="2" fill="none" opacity="0.6"/>`, accent, soft, accent)
	}
}

// ---------- who may change what ----------

func needUser(w http.ResponseWriter, r *http.Request) (*User, bool) {
	u := currentUser(r)
	if u == nil {
		guestRedirect(w, r)
		return nil, false
	}
	return u, true
}

type StyleOption struct{ Slug, Name string }

var avatarFrames = []StyleOption{
	{"", "No frame"}, {"neon", "Neon"}, {"purple", "Purple glow"},
	{"pink", "Pink glow"}, {"gold", "Golden"}, {"silver", "Silver"},
	{"cyber", "Cyberpunk"}, {"pulse", "Animated pulse"},
}

var avatarBanners = []StyleOption{
	{"", "No banner"}, {"aurora", "Aurora"}, {"ember", "Ember"},
	{"ocean", "Ocean"}, {"forest", "Forest"}, {"dusk", "Dusk"}, {"candy", "Candy"},
}

// ---------- the avatar studio ----------

func studioPageHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := needUser(w, r)
	if !ok {
		return
	}
	data := PageData{
		User:         u,
		Theme:        "base",
		Nav:          "settings",
		AvatarGroups: avatarGallery(),
		Frames:       avatarFrames,
		Banners:      avatarBanners,
	}
	switch r.URL.Query().Get("ok") {
	case "upload":
		data.Saved = "Your new avatar is ready!"
	case "builtin":
		data.Saved = "Avatar updated."
	case "remove":
		data.Saved = "Your picture was removed."
	case "style":
		data.Saved = "Your look was saved."
	}
	if e := r.URL.Query().Get("err"); e != "" {
		data.Error = e
	}
	render(w, "avatar-studio.html", data)
}

func avatarUploadHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := needUser(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, avatarMaxBody)
	if err := r.ParseMultipartForm(avatarMaxBody); err != nil {
		http.Redirect(w, r, "/avatar-studio?err="+urlMsg("That picture is too large. Please keep it under 4 MB."), http.StatusSeeOther)
		return
	}
	file, _, err := r.FormFile("picture")
	if err != nil {
		http.Redirect(w, r, "/avatar-studio?err="+urlMsg("Please choose a picture first."), http.StatusSeeOther)
		return
	}
	defer file.Close()

	data, err := shrinkToAvatar(file)
	if err != nil {
		http.Redirect(w, r, "/avatar-studio?err="+urlMsg(err.Error()), http.StatusSeeOther)
		return
	}
	if err := saveAvatar(u.ID, "upload", "", data); err != nil {
		log.Println("avatar upload:", err)
		http.Redirect(w, r, "/avatar-studio?err="+urlMsg("We could not save that picture. Please try again."), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/avatar-studio?ok=upload", http.StatusSeeOther)
}

func avatarBuiltinHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := needUser(w, r)
	if !ok {
		return
	}
	id := strings.TrimSpace(r.FormValue("avatar"))
	if _, ok := builtinAvatarSVG(id); !ok {
		http.Redirect(w, r, "/avatar-studio?err="+urlMsg("That avatar does not exist."), http.StatusSeeOther)
		return
	}
	if err := saveAvatar(u.ID, "builtin", id, nil); err != nil {
		log.Println("avatar builtin:", err)
		http.Redirect(w, r, "/avatar-studio?err="+urlMsg("We could not save that avatar."), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/avatar-studio?ok=builtin", http.StatusSeeOther)
}

func avatarRemoveHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := needUser(w, r)
	if !ok {
		return
	}
	if err := saveAvatar(u.ID, "", "", nil); err != nil {
		log.Println("avatar remove:", err)
	}
	http.Redirect(w, r, "/avatar-studio?ok=remove", http.StatusSeeOther)
}

func avatarStyleHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := needUser(w, r)
	if !ok {
		return
	}
	frame := r.FormValue("frame")
	banner := r.FormValue("banner")
	calm := r.FormValue("calm") == "on"

	goodFrame := false
	for _, f := range avatarFrames {
		if f.Slug == frame {
			goodFrame = true
		}
	}
	goodBanner := false
	for _, b := range avatarBanners {
		if b.Slug == banner {
			goodBanner = true
		}
	}
	if !goodFrame || !goodBanner {
		http.Redirect(w, r, "/avatar-studio?err="+urlMsg("That option is not available."), http.StatusSeeOther)
		return
	}
	if err := saveProfileStyle(u.ID, frame, banner, calm); err != nil {
		log.Println("avatar style:", err)
	}
	http.Redirect(w, r, "/avatar-studio?ok=style", http.StatusSeeOther)
}

func urlMsg(s string) string {
	r := strings.NewReplacer(" ", "+", "&", "and", "?", "", "#", "")
	return r.Replace(s)
}

// ---------- profile ----------

func displayNameOf(u *User) string {
	if u == nil {
		return ""
	}
	if strings.TrimSpace(u.DisplayName) != "" {
		return u.DisplayName
	}
	return u.FullName
}

func profilePageHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := needUser(w, r)
	if !ok {
		return
	}
	data := PageData{
		User:         u,
		Theme:        "base",
		Nav:          "profile",
		Achievements: achievementsFor(*u),
	}
	if r.URL.Query().Get("ok") == "saved" {
		data.Saved = "Profile saved."
	}
	render(w, "profile.html", data)
}

func profileEditHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := needUser(w, r)
	if !ok {
		return
	}
	render(w, "profile-edit.html", PageData{
		User:        u,
		Theme:       "base",
		Nav:         "profile",
		DisplayName: u.DisplayName,
		Bio:         u.Bio,
	})
}

func profileEditSaveHandler(w http.ResponseWriter, r *http.Request) {
	u, ok := needUser(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	name := strings.TrimSpace(r.FormValue("display_name"))
	bio := strings.TrimSpace(r.FormValue("bio"))

	if len([]rune(name)) > 40 {
		render(w, "profile-edit.html", PageData{User: u, Theme: "base", Nav: "profile",
			DisplayName: name, Bio: bio, Error: "A display name can be at most 40 characters."})
		return
	}
	if len([]rune(bio)) > 300 {
		render(w, "profile-edit.html", PageData{User: u, Theme: "base", Nav: "profile",
			DisplayName: name, Bio: bio, Error: "A bio can be at most 300 characters."})
		return
	}
	if err := saveProfileText(u.ID, name, bio); err != nil {
		log.Println("profile save:", err)
		render(w, "profile-edit.html", PageData{User: u, Theme: "base", Nav: "profile",
			DisplayName: name, Bio: bio, Error: "We could not save that. Please try again."})
		return
	}
	http.Redirect(w, r, "/profile?ok=saved", http.StatusSeeOther)
}

// ---------- achievements ----------
// Everything here is worked out from what the account has actually done:
// real visits in its history and how old the account is. Nothing is invented,
// and a badge only shows as earned when its condition is really true.

type Achievement struct {
	ID     string
	Name   string
	Desc   string
	Icon   string
	Earned bool
	Have   int
	Need   int
}

func achievementsFor(u User) []Achievement {
	topics := map[string]bool{}
	count := 0
	for _, h := range history {
		if h.UserID != u.ID || h.LanguageSlug == "" {
			continue
		}
		if !topics[h.LanguageSlug] {
			topics[h.LanguageSlug] = true
			count++
		}
	}
	visited := func(slugs ...string) bool {
		for _, s := range slugs {
			if topics[s] {
				return true
			}
		}
		return false
	}
	days := int(time.Since(u.CreatedAt).Hours() / 24)
	profileDone := u.AvatarKind != "" && strings.TrimSpace(u.Bio) != "" && strings.TrimSpace(u.DisplayName) != ""

	return []Achievement{
		{ID: "explorer", Name: "Pathfinder Explorer", Icon: "🧭", Desc: "Visit three different topics", Earned: count >= 3, Have: count, Need: 3},
		{ID: "manga", Name: "Manga Explorer", Icon: "📚", Desc: "Open a manga or manhwa topic", Earned: visited("read-manga", "read-manhwa", "manga", "manhwa"), Have: b2i(visited("read-manga", "read-manhwa", "manga", "manhwa")), Need: 1},
		{ID: "anime", Name: "Anime Explorer", Icon: "🎬", Desc: "Open the anime topic", Earned: visited("anime"), Have: b2i(visited("anime")), Need: 1},
		{ID: "books", Name: "Book Lover", Icon: "📖", Desc: "Open the books topic", Earned: visited("books"), Have: b2i(visited("books")), Need: 1},
		{ID: "movies", Name: "Movie Enthusiast", Icon: "🍿", Desc: "Open cartoons and movies", Earned: visited("cartoons-movies"), Have: b2i(visited("cartoons-movies")), Need: 1},
		{ID: "profile", Name: "All Set Up", Icon: "✨", Desc: "Add an avatar, a display name and a bio", Earned: profileDone, Have: b2i(profileDone), Need: 1},
		{ID: "veteran", Name: "PathFinder Veteran", Icon: "🏅", Desc: "Be here for 30 days", Earned: days >= 30, Have: days, Need: 30},
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// builtinPreviewHandler serves one of our own drawings for the gallery grid.
func builtinPreviewHandler(w http.ResponseWriter, r *http.Request) {
	svg, ok := builtinAvatarSVG(r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=604800")
	w.Write(svg)
}
