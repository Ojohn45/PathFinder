# PathFinder in a small container.
# Build:  docker build -t pathfinder .
# Run:    docker run -d --name pathfinder -p 8080:8080 \
#           --env-file .env -v pathfinder-data:/app/storage pathfinder

# The Go version here must be at least the version in go.mod. Our dependencies
# (golang.org/x/image) now ask for Go 1.26, and a 1.24 image cannot build them,
# which is what broke the deploy on 26 November.
FROM golang:1.26-alpine AS build
WORKDIR /src

# Everything, including the vendor/ folder, so the build needs no internet.
COPY . .

# -mod=vendor means Go reads the dependencies from vendor/ instead of
# downloading them. Nothing can go missing and no proxy can be unreachable.
RUN CGO_ENABLED=0 go build -mod=vendor -trimpath -ldflags "-s -w" -o /out/pathfinder .

FROM alpine:3.20
# su-exec lets the entrypoint fix ownership of the mounted data disk, then drop
# from root to the app user. A host-mounted volume arrives owned by root, so a
# plain USER app image could not write users.json on Render or Docker.
RUN apk add --no-cache su-exec && adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/pathfinder ./pathfinder
COPY templates ./templates
COPY static ./static
RUN mkdir -p /app/storage && chown -R app:app /app
ENV PORT=8080
EXPOSE 8080
VOLUME ["/app/storage"]
CMD ["sh", "-c", "chown -R app:app /app/storage 2>/dev/null || true; exec su-exec app ./pathfinder"]
