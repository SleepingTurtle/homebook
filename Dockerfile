# Build stage
FROM golang:1.22-alpine AS builder

# Version build arguments
ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG GIT_COMMIT=unknown

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-X homebooks/internal/version.Version=${VERSION} \
              -X homebooks/internal/version.BuildTime=${BUILD_TIME} \
              -X homebooks/internal/version.GitCommit=${GIT_COMMIT} \
              -s -w" \
    -o homebooks ./cmd/server

# Runtime stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/homebooks .

# Create data directory
RUN mkdir -p /data

ENV HOMEBOOKS_DB_PATH=/data/homebooks.db
ENV PORT=8080

EXPOSE 8080

CMD ["./homebooks"]
