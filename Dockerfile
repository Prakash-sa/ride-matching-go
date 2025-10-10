# Build stage
FROM golang:1.22 AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/ride-matching ./cmd/server

# Run stage
FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/ride-matching /app/ride-matching
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/ride-matching"]
