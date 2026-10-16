# ---------- Stage 1: build ----------
# Build with an image that has the full Go toolchain
FROM golang:1.27-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /taskapi .

# ---------- Stage 2: run ----------
FROM alpine:3.20

COPY --from=builder /taskapi /taskapi
USER nobody
EXPOSE 8080
ENTRYPOINT ["/taskapi"]
