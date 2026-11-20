# PathFinder in a small container.
# Build:  docker build -t pathfinder .
# Run:    docker run -d --name pathfinder -p 8080:8080 \
#           --env-file .env -v pathfinder-data:/app/storage pathfinder

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/pathfinder .

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
