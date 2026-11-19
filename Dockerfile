# PathFinder in a small container.
# Build:  docker build -t pathfinder .
# Run:    docker run -d --name pathfinder -p 8080:8080 \
#           --env-file .env -v pathfinder-data:/app/storage pathfinder

FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/pathfinder .

FROM alpine:3.20
RUN adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/pathfinder ./pathfinder
COPY templates ./templates
COPY static ./static
RUN mkdir -p /app/storage && chown -R app:app /app
USER app
ENV PORT=8080
EXPOSE 8080
VOLUME ["/app/storage"]
CMD ["./pathfinder"]
