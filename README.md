# PathFinder

Find your way to the right resource. A Go web app that sends people to hand-picked places worth their time: programming,
entertainment, learning, lifestyle, creating and working, plus reading manga, manhwa
and comix. Eight sections, each with its own colour, and 71 programming languages
with fifteen places each.
No database, no framework. Plain `net/http` and JSON files.

## The name and the logo

PathFinder, because the whole point is finding the one right place instead of opening
twenty tabs. The logo is the word PATHFINDER in a dark blue wash with a route line
drawing itself underneath and an arrow at the end pointing forward: it wipes in from
the left, holds, then dissolves, on a seven second loop. Corner brackets, a single
scan sweep and a brief glitch keep it techy without shouting.

## Run it

```bash
cp .env.example .env     # then fill it in
go run .
```

Open http://localhost:8080

## What you need

- Go 1.26 or newer (the `go` line in `go.mod`).
- A `.env` file next to `main.go` with at least `ADMIN_PASSWORD` and the SMTP details
  for welcome and invitation emails. See `.env.example` for Gmail and Brevo setups.

## The head admin

There is **no built-in password**. Set `ADMIN_PASSWORD` (in `.env` on your computer, or in
Render > Environment) and the head admin is created on start-up:

| field    | value                                         |
|----------|-----------------------------------------------|
| name     | `ADMIN_FULLNAME` (default: Odumu John)        |
| username | `ADMIN_USERNAME` (default: Ken)               |
| email    | `ADMIN_EMAIL`                                 |
| phone    | `ADMIN_PHONE` (optional, never written in code) |
| password | your `ADMIN_PASSWORD`                         |

If `ADMIN_PASSWORD` is not set, no head admin is created. Log in on the normal `/login`
page. To change the password later, change `ADMIN_PASSWORD` and restart: the new value
replaces the old one. Passwords are stored as bcrypt hashes, never as plain text.

## Storage

Everything lives in `storage/` as plain JSON. Back it up by copying the folder.

| file           | holds                                        |
|----------------|----------------------------------------------|
| `users.json`   | accounts, roles, online status, invite state |
| `history.json` | which places each person has opened          |
| `invites.json` | invitations and the tokens behind the links  |

## How the pages fit together

- `/` landing page, then `/login`, `/signup`, `/join?token=...`
- `/home` the four explanation boxes, the Browse button and every fraction
- `/browse` every fraction and topic in one list
- `/fraction/<slug>` one fraction in its own colour
- `/topic/<slug>` the places for one topic, each opening in a new tab

### Programming works differently on purpose

The topic boxes always show, but clicking one explains the three ways in:

1. **Watch documentaries**: pick a language, then see where to watch films about it.
2. **Straight to learning**: pick a language, then get its fifteen places to learn it.
3. **Docs**: pick a language, then read its documentation.

`/fraction/programming`, `/fraction/programming/learning`,
`/fraction/programming/docs`, `/fraction/programming/documentaries`.

Read Manga and Read Manhwa skip the topic step: their places are all on the
fraction page at once.

## Who is the admin

There is no separate admin login page. The head admin signs in on the normal Log In
form using the registered phone number (08012345678 by default, set with
`ADMIN_PHONE`), their username (Ken) or their email. Admins land on the dashboard
automatically.

## History and search

`/history` shows every place a visitor opened, newest first, grouped by day.

Search lives inside five sections only (Programming, Entertainment, Read Manga,
Read Manhwa, Read Comix) and never on the landing, home or Browse pages. The landing,
home and Browse pages have none by design.

## Sending email

No `.env` file is required. Start the app, log in as the head admin, open
**Admin > Email Settings**, and type the sending address plus an app password
(Gmail: a 16 letter App Password; Brevo: its SMTP key). Press **Save and send a test
email**. The details are stored in `storage/settings.json`, which git ignores.

If you prefer environment variables (`SMTP_USER`, `SMTP_PASS`, `SMTP_HOST`,
`SMTP_PORT`, `SMTP_FROM`), they take priority over the page. See `.env.example`.

## Going live

For Render and `pathfinder.app`, see **RENDER.md**. For your own server, see
**DEPLOY.md**. Short version: buy a domain, rent the smallest Ubuntu box you can
find, point the DNS at it, fill in `.env`, then `docker compose up -d --build`. Caddy
in the compose file fetches the HTTPS certificate for you. The `pathfinder-data`
volume keeps `storage/` alive across rebuilds, which matters because the JSON files
are the whole app's memory.

## Files

```
main.go        routes, page data, the back button rules
content.go     every fraction, topic and place, in one place
data.go        the place lists for each topic
db.go          JSON storage: load, save, history
auth.go        signup, login, logout, sessions
admin.go       admin dashboard, check users, invites, emails
email.go       SMTP: welcome mail, invitation mail
prog.go        the programming room: learning, docs, documentaries
templates/     one file per page, partials.html holds the shared bits
static/        style.css, one theme block per fraction
storage/       the JSON files (created on first run if missing)
render.yaml    Render Blueprint (service, disk, variables)
RENDER.md      Render walkthrough, step by step
Dockerfile     builds a small image
docker-compose.yml   app + Caddy, HTTPS included
Caddyfile      put your domain in here
pathfinder.service   systemd unit if you skip Docker
DEPLOY.md      the step by step for going live
```

## Design rules baked in

- Every page is filled edge to edge with its own colour.
- A Back button sits top left on every page except the home page and the admin
  dashboard, which are home bases.
- Language boxes are large with bold centred names; site boxes are half that size.
- Every outgoing link opens in a new tab.
- Logging out asks YES or NO. YES says thank you, NO keeps you logged in.
