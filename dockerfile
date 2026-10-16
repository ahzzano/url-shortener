FROM golang:1.26.3-alpine AS builder

WORKDIR /app

COPY go.mod ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /main .

FROM alpine:3.21
WORKDIR /
COPY --from=builder /main /main
EXPOSE 8080
ENTRYPOINT [ "/main" ]
