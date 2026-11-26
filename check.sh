#!/bin/sh
# Run in your PathFinder repo before pushing.
missing=0
for f in go.mod go.sum Dockerfile render.yaml main.go db.go auth.go admin.go \
  avatar.go content.go data.go pg.go templates/home.html templates/landing.html \
  templates/landingbody.html templates/avatar-studio.html templates/profile.html \
  templates/profile-edit.html templates/partials.html templates/topic.html \
  templates/search.html static/style.css static/img/sections.jpg \
  static/img/places.jpg static/img/open.jpg static/img/return.jpg static/img/phone.jpg
do
  [ -f "$f" ] || { echo "MISSING: $f"; missing=1; }
done
[ "$missing" -eq 0 ] && echo "All required files are here." || { echo ""; echo "Push the whole extracted folder, not hand-picked files."; exit 1; }
for f in go.mod go.sum; do
  grep -q "golang.org/x/image" "$f" || { echo "$f is an old copy (no golang.org/x/image)."; exit 1; }
done
echo "go.mod and go.sum include the image library."
if command -v go >/dev/null 2>&1; then
  echo "Checking that it compiles..."
  go build -o /dev/null . && echo "Compiles cleanly. Safe to push." || { echo "Does not compile - push the whole folder again."; exit 1; }
fi
