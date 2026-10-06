FROM node:22-alpine AS web
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci
COPY index.html tsconfig.json tsconfig.app.json vite.config.ts ./
COPY src ./src
COPY public ./public
RUN npm run build

FROM golang:1.24-alpine AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /r34watch ./cmd/r34watch

FROM alpine:3.22
RUN apk add --no-cache aria2 wget ffmpeg ca-certificates tzdata
WORKDIR /app
COPY --from=backend /r34watch /usr/local/bin/r34watch
COPY --from=web /src/web/dist /app/web/dist
COPY openapi.yaml /app/openapi.yaml
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh
ENV R34WATCH_ADDR=:8787 R34WATCH_DB=/data/r34watch.db R34WATCH_WEB=/app/web/dist R34WATCH_MEDIA_ROOT=/media
VOLUME ["/data","/media"]
EXPOSE 8787
ENTRYPOINT ["docker-entrypoint.sh"]
