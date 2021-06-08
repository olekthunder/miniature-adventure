FROM golang:1.16 AS builder
RUN mkdir /app
WORKDIR /app/
COPY main.go /app/
ENV GO111MODULE=off
RUN CGO_ENABLED=0 GOOS=linux go build -o main 

FROM alpine:latest AS production
COPY --from=builder /app/main .
CMD ["./main"]