FROM node:24-alpine AS frontend-builder
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS backend-builder
WORKDIR /build
COPY backend/ ./
RUN go build cmd/go-poker/main.go

FROM alpine:3.24
RUN apk --no-cache add ca-certificates && \
    adduser -S -D -H -h /app appuser
USER appuser
WORKDIR /app

COPY --from=backend-builder /build/main ./

COPY --from=frontend-builder /app/web/out /app/out

CMD ["./main"]
