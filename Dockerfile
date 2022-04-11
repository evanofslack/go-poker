FROM golang:1.14.9-alpine AS builder
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN go build

FROM alpine
RUN adduser -S -D -H /app appuser
USER appuser
COPY --from=builder /build/go-poker /app/
WORKDIR /app
CMD ["./go-poker"]
