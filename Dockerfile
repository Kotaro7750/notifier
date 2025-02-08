FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .
RUN go build -o main .

FROM alpine:3.21

WORKDIR /app

COPY --from=builder /app/main .

ENTRYPOINT ["./main"]
