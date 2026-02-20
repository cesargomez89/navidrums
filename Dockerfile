FROM docker.io/library/golang:1.25.5-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o navidrums ./cmd/server

FROM docker.io/library/alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/navidrums .

EXPOSE 8080

VOLUME ["/downloads"]

CMD ["./navidrums"]
