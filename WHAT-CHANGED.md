# Redirect - fractions upgrade

## New pages
- `/home` - four explanation boxes, the Browse button, the five fractions, recently visited.
- `/browse` - every fraction as a big box (guests get the sign-up popup).
- `/fraction/{slug}` - one fraction in its own colour, with its topics.
- `/topic/{slug}` - the 15 places for that topic.

## Still working
- `/go/{slug}/{resourceID}` saves history and redirects.
- `/languages` and `/language/{slug}` now redirect to the new pages, so old links keep working.
- Signup, login, welcome email, admin, invites, Check Users and the logout confirmation are unchanged.

## Colours
Every page carries `data-theme` on `<body>`: base, prog, ent, learn, life, work.
The whole palette lives in one place, in the THEME LAYER section of `static/style.css`.

## The logo
`{{template "logo" .}}` in `templates/partials.html`. It sits in the top bar of every page,
on the landing hero, and on the login / signup / admin-login pages.
Size is set with `--logo-size` on the surrounding element.

## Where to add things
- A new language: add an entry to `programmingTopics` in `data.go`.
- A new topic in another fraction: add it to `otherTopics` in `content.go`, then list it
  under that fraction in the `fractions` slice.
- Box sizes: `--box-size` and `--site-h` at the top of `static/style.css`.
