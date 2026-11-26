# What changed, latest first

## v28 - the deploy that failed on 26 November
- Cause: the Dockerfile built with golang:1.24-alpine, but golang.org/x/image
  (added for avatars) declares go 1.26.0. A 1.24 toolchain cannot build it, so the
  build stopped before the app ever started.
- Fix: the build image is now golang:1.26-alpine, matching go.mod.
- Dependencies are vendored (vendor/, 75 files, 760KB) and the build uses
  -mod=vendor, so it needs no internet and no module proxy at build time.
- Added .dockerignore so storage/, .git and zips stay out of the image.

## v27 - avatars and profiles
- Every account can now have a picture: uploaded (JPG, PNG, WebP) or one of 60
  original drawings in ten styles.
- Uploads are cropped in the browser, then shrunk to a 256x256 square JPEG on the
  server and stored in Postgres, so they survive redeploys.
- New: /avatar-studio (upload, gallery, frames, banners, reduced motion),
  /profile and /profile/edit.
- Eight frames, six banners, seven achievements worked out from real history.
- The account button in the sidebar opens a menu: profile, edit, avatar, settings, log out.

## v25 - accounts that survive, and a site that works on a phone
### Data
- `pg.go` (new): Postgres storage. Set DATABASE_URL and users, history, invites,
  the email log, mail settings and sessions all live in a real database with
  tables created automatically at startup.
- Logout was never the problem: it only deletes the session cookie. The data was
  in storage/*.json, and Render wipes that folder on redeploy, restart and sleep.
- Sessions are saved in Postgres, so a redeploy no longer signs everyone out.
- Passwords were already bcrypt; unchanged and still hashed.
- With no DATABASE_URL the app still uses JSON files, so `go run .` on a laptop
  behaves exactly as before.

### Phones
- Viewport tag added to the 11 pages that were missing it.
- Menu is a drawer under 900px: closed at startup, floats over the page with a
  dim backdrop, closes on an outside tap, on Escape, or on choosing a link.
- Page starts 66px down on phones so the menu button never sits on the text.
- Category cards: one per row at 150px tall on a phone, two across on a tablet,
  the grid on desktop.
- Admin buttons stack full width; tap targets are at least 44px; long tables
  scroll inside their own box. No page scrolls sideways at 360, 390, 412, 768
  or 1024px.

## v24 - every sport
- The Sports section now lists 48 sports, each opening its own watch page.
- Six already existed (Football, Basketball, Tennis, Boxing & MMA, Motorsport, Cricket).
- Forty-two were added with their real, verified sources only: no padding with the
  same broadcasters. Sports with fewer than 15 real places show fewer, and say so.
- Every place is verified (2xx/3xx) or confirmed live but bot-blocked, and each has a
  description under 60 characters.

## v23 - Sports, Loans and Read Novels
- **Sports** section with six sports to pick from (Football, Basketball, Tennis,
  Boxing & MMA, Motorsport, Cricket), 15 places to watch each. No betting sites, as
  agreed: watching only.
- **Loans** section, 20 licensed lenders and credit providers, Nigerian first, with the
  Central Bank register so anyone can check a lender is real.
- **Read Novels** section with five kinds: Historical, Sci-Fi, Fiction, Based on a True
  Story, and Romance. Romance stands in for the 18+ erotica you asked for, because this
  project's own rules say no adult content; say the word if you want that changed.
- Clothing & Accessories now says "Where to see clothing & accessories" instead of
  "Where to read". The heading is chosen per section in `sitesLabelFor`.
- Three new dark themes: `sports` (dark orange), `loan` (dark teal-green) and `novel`
  (deep violet).
- `Topic.Want` added so a topic can ask for a different count; Loans uses 20, the rest
  stay at 15, and the test honours it.
- Gutenberg and LibriVox were left out: both refused every check from here (503 and 522),
  so they could not be verified. Archive.org copies of LibriVox are used instead.

## v22 - Phase 1 sections
- Three new sections: **Exams & Scholarships** (exams), **Jobs & Careers** (jobs) and
  **Developer Tools** (devtools), five topics each, fifteen places each: 225 in total.
- Each has its own dark theme: dark gold-olive, dark slate-steel, and dark graphite with
  an orange accent. Readable text, same animation style, no new libraries.
- These three sections have no search bar on purpose. The search bar answers "where can I
  watch/read/listen to this title" and returns outside search links; for a fixed list of
  exam boards, job boards and developer tools it would only produce a search engine link.
- `content_test.go` added: fifteen places per topic, unique IDs and URLs, https only,
  unique section slugs and themes, and a CSS block for every theme.
- Fixed a real bug the tests found: the C++ topic listed LearnCpp twice and had two
  entries for the C++ Core Guidelines, so it was short of distinct places. Replaced with
  cpp.sh and CppCon.

## v21
- **The sidebar folds away and comes back**, like the one in your recording. A button
  at the top of the sidebar folds it; a button in the corner brings it back; Ctrl+Shift+B
  does the same from the keyboard. On a computer it remembers your choice between pages.
  On a phone it starts closed and slides over the page.
- The name at the top of the sidebar is plain text, **PathFinder**, with no animation.
  The animated wordmark stays on the landing page.
- **The strip down the right edge is gone.** It was the page background painted twice:
  once full width and once only as wide as the 1000px content column, so a darker band
  showed beside it. The colour is now painted once, edge to edge.
- Scrollbars are hidden so no extra track is drawn at the right.

## v20
- **The whole menu now lives in a left sidebar**, the way a working app does it:
  the PathFinder wordmark at the top, then Home, Browse, History and Admin (admins
  only), and pinned to the bottom left, **Settings** and **Log out**. The current page
  is highlighted.
- The Back button is the only thing that stayed out of the sidebar: it sits at the top
  left of the page content, so it always means "go back from here".
- On a phone the sidebar slides in from the left behind a small menu button, with a
  dark scrim, and closes on Escape or a tap outside. Settings stays at the bottom.
- Content is offset so nothing hides behind the sidebar, and the wordmark is centred in
  the sidebar on Home and Admin.

## v19
- **No public admin door.** The "Log In as Admin" button is gone from the landing
  page and the Log In page, and `/admin/login` now just sends you to the normal login
  page. Admins sign in on the same form as everybody else.
- **Log in with the registered number.** The head admin account carries the phone
  number 08127758529 (change it with `ADMIN_PHONE`), and the login box accepts a
  username, an email address or a phone number. The number can be typed as
  08127758529, 0812 775 8529 or +2348127758529. Signing in with it lands on the admin
  dashboard.
- The head admin is created at start-up as Odumu John / Ken with the password
  49321111@John unless `ADMIN_PASSWORD` says otherwise.
- The download no longer ships `storage/*.json`. Earlier builds carried test accounts,
  which meant a fresh install inherited an old head admin password. `storage/` now
  arrives empty and is created on first run.
- **Search results now carry the search.** Every link on a search results page embeds
  the term in the destination's own search URL, so the site opens already showing what
  was searched. Path-style searches (Spotify, Deezer, PlayStation) now encode spaces
  as %20 instead of +, which is what those sites expect.

## v18
- **History page** at `/history`, linked from the top bar and from "Recently visited"
  on the home page. Every place a visitor opens is listed newest first, grouped into
  Today / Yesterday / the date, with the section, the topic and the time, and the
  name is a link straight back to that place.
- **Search inside a section, and nowhere else.** The landing page, the home page and
  Browse have no search bar. Only these five sections carry one:
  - Programming searches the languages themselves (nicknames work: js, ts, golang, py, c++, c#).
  - Entertainment searches a film, anime, cartoon, song or game title and answers in
    three rows: **Watch it**, **Listen to it**, **Play it**, each with the real
    platforms (JustWatch, Netflix, Prime Video, Crunchyroll, Spotify, Audiomack,
    Steam, PlayStation, Xbox, Nintendo and more).
  - Read Manga, Read Manhwa and Read Comix search a title and answer with where to
    read it, official publishers first.
- Learning, Lifestyle and Create & Work deliberately have no search bar.
- Search results open in a new tab, so PathFinder stays behind them.

## v17
- **Programming went from 10 languages to 71.** The ten hand-curated ones stay
  (Python, Go, CSS, HTTP, HTTPS, Java, C++, PHP, SQL, JavaScript); 61 more were added:
  C, C#, Rust, Ruby, Kotlin, Swift, Scala, Dart, Elixir, Erlang, Haskell, Clojure, F#,
  OCaml, Lua, R, Julia, MATLAB, Perl, Bash, PowerShell, Assembly, Objective-C, Groovy,
  Visual Basic, Fortran, COBOL, Ada, Prolog, Scheme, Racket, Common Lisp, Crystal, Nim,
  Zig, D, Elm, Solidity, HTML, TypeScript, Sass, GraphQL, Verilog, VHDL, Tcl,
  Smalltalk, Forth, APL, Processing, Wolfram Language, SAS, ABAP, Apex, Delphi,
  Scratch, Vyper, Move, Gleam, Cairo, Odin and V.
- Every language still carries exactly fifteen places, and all three doors work for
  all of them: docs, straight to learning, and documentaries.
- Each new language leads with its real documentation, then dependable destinations:
  Wikipedia, Stack Overflow, Rosetta Code, DevDocs, Exercism where a track exists,
  Learn X in Y Minutes, freeCodeCamp, Coursera, Udemy, YouTube, GitHub, Codecademy
  and Reddit.
- All 911 links in the new list were checked by machine; the twelve that did not
  answer were replaced.
- Added a search box above the language grid, so 71 boxes stay usable: type "rust"
  and the rest step aside.

## v16
- **No .env file needed any more.** The head admin gets a new **Admin > Email
  Settings** page: type the mail server, the sending address and the app password,
  press Save, and invitations and welcome emails start sending. The details live in
  `storage/settings.json` (mode 600, inside the folder git already ignores), so they
  never reach GitHub. A "Save and send a test email" button proves it works.
- Environment variables still win when they exist, so Render, Docker and a VPS can
  keep controlling their own secrets. The page says so when that is the case.
- Invite links no longer hardcode `localhost`. If `BASE_URL` and Render's own URL are
  both absent, the link borrows the address the admin is browsing from, so an invite
  made from a phone on the same Wi-Fi produces a link that phone can actually open.
- The app still binds `:PORT` on all interfaces, so `http://<your-computer-ip>:8080`
  works from a phone on the same network.
- All the "add SMTP_USER and SMTP_PASS to .env" messages now point at the Email
  Settings page instead.

## v15
- "Fractions" are now "sections", everywhere: the word, the URLs (`/section/programming`
  instead of `/fraction/programming`), the code and the colours' class names.
- New section: **Read Comix**, fifteen places to read comics, in burnt orange. Like
  Manga and Manhwa it skips the topic step and shows all fifteen at once.
- Back button removed from the landing page, the Log In page and the Sign Up page.
  It stays on every other page except the home page and the admin dashboard.
- Copy updated: the site now says eight sections, and the home and landing text
  mentions manga, manhwa and comix.
- Added `render.yaml` and `RENDER.md` for deploying to Render at pathfinder.app.
- The Docker image now fixes ownership of the mounted data disk before starting, so
  the app can still write `users.json` when a host mounts a volume at `/app/storage`.

## v14
- Renamed from Redirect to PathFinder everywhere a person can see it: page titles,
  headings, emails, the invitation text, the admin screens and the README.
- New logo: PATHFINDER in a dark blue wash, with a route line that draws itself under
  the word and an arrow at the end pointing forward. It wipes in from the left, holds,
  then dissolves on a seven second loop. Corner brackets, one scan sweep, a brief
  glitch. Stillness for anyone who asks for reduced motion.
- New tagline: "Find your way to the right resource."
- New email banner to match the new name.
- The app now reads `$PORT`, so hosting platforms that assign a port work without a
  code change. Locally it still uses 8080.
- Added Dockerfile, docker-compose.yml, Caddyfile, pathfinder.service and DEPLOY.md:
  a domain, a small server and one command gets it live with HTTPS.
- The admin invitation mail said "10 of the best places"; it says 15 now.

## v13 - the latest
- Read Manga and Read Manhwa show all fifteen places straight on the fraction page,
  with no topic step in between. Each link opens in a new tab and is recorded.
- Other fractions keep their topic boxes.

## v12
- The Back button is gone from the home page and the admin dashboard. It is still on
  every other page, including the admin sub-pages.

## v11
- A Back button at the top left of every page, returning to the previous page.
- Removed a leftover block from the old anime intro that painted a black vignette over
  every page and drew a dark red line under the top bar. That was what made the pages
  look dark instead of coloured.
- Every page and fraction now fills edge to edge in its own colour, including outside
  the content column and on overscroll. Cards, site boxes and the top bar take the page
  colour instead of near-black or white.

## v10
- Programming topics still show normally, but clicking one now brings up the message
  asking the visitor to pick Watch documentaries, Straight to learning or Docs.
- Each of the three asks for the language first, then shows that language's places.
- Straight to learning added: fifteen curated places per language.
- Fixed the language chooser pages rendering empty (an empty language counted as
  filled in the template).

## v9
- The KAI anime intro is gone. The logo is a quiet techy wordmark: REDIRECT in a dark
  blue gradient with a scan sweep, corner brackets and a brief glitch, appearing then
  dissolving on a seven second loop.
- Read Manga and Read Manhwa added, fifteen sites each, with their own colours.
- Every redirect link opens in a new tab.

## v8 and earlier
- Signup collects an email and sends a welcome mail. Login and admin login forms work.
- Head admin Ken (Odumu John) auto-creates on first start, so the admin page and
  invites work immediately.
- Logout asks YES or NO, YES says thank you, NO keeps you logged in.
- Large language boxes with bold centred names, site boxes at half that size.
- Storage is JSON in `storage/`, never SQLite.
