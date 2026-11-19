# What changed, latest first

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
