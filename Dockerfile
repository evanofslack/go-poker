FROM golang:1.18.2-alpine as builder
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN go build

FROM alpine
RUN adduser -S -D -H -h /app appuser
USER appuser
COPY --from=builder /build/go-poker /app/
WORKDIR /app
CMD ["./go-poker"]
