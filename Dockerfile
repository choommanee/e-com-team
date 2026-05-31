# ---- build ----
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

# ---- runtime ----
FROM alpine:3.20
RUN adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/server /app/server
# Templates + static assets are embedded in the binary; output dir for images:
RUN mkdir -p /app/output && chown -R app /app
USER app
EXPOSE 8080
ENV PORT=8080 OUTPUT_DIR=/app/output
ENTRYPOINT ["/app/server"]
