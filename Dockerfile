FROM golang:1.24.4-alpine AS builder

# Install build dependencies for CGO and SQLite
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o auto-focus-cloud main.go

FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite

WORKDIR /root/

# Create storage directory for SQLite database
RUN mkdir -p storage/data

COPY --from=builder /app/auto-focus-cloud .

EXPOSE 8080

CMD ["./auto-focus-cloud"]
