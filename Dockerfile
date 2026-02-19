FROM docker.io/library/golang:1.25.5-alpine AS builder

WORKDIR /app

# change PROVIDER_URL to your hifi-api url
# changing NAVIDRUMS_USERNAME and NAVIDRUMS_PASSWORD is recommended
ENV PORT=8080 \
  DB_PATH=navidrums.db \
  DOWNLOADS_DIR=/downloads \
  PROVIDER_URL=https://hifi-api.com \
  QUALITY=LOSSLESS \
  LOG_LEVEL=info \
  LOG_FORMAT=text \
  NAVIDRUMS_USERNAME=navidrums \
  NAVIDRUMS_PASSWORD=admin

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
