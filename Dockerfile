# syntax=docker/dockerfile:1

#
# BUILD stage
#
FROM golang:1.17-alpine as build

WORKDIR /app

# Download dependencies first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /bin/go-load-tester

#
#  FINAL stage
#
FROM alpine:3.14 as final
RUN apk --no-cache add bash
WORKDIR /
COPY --from=build /bin/go-load-tester /bin/go-load-tester
COPY _Documents _Documents
COPY templates templates
COPY static static

ENTRYPOINT ["/bin/go-load-tester"]
