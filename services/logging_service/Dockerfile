FROM golang:1.16 AS builder
RUN mkdir /app
COPY go.mod go.sum /app/
WORKDIR /app/
RUN go mod download
COPY main.go /app/
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./...

FROM alpine:latest AS production
COPY --from=builder /app/main .
CMD ["./main"]