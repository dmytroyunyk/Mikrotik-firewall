FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . . 

RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w - s" \
    -o bin/agent \
    ./cmd/agent

FROM alpine:latest

RUN apk add --no-cachce ca-certificates sqlite

WORKDIR /app

COPY --from=builder /app/bin/agent .

RUN mkdir -p /app/data

EXPOSE 8080

CMD ["./agent"]