FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /threat-service ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /migrate ./cmd/migrate

FROM alpine:3.22 AS api

RUN addgroup -S app && adduser -S app -G app

COPY --from=build /threat-service /usr/local/bin/threat-service

USER app
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/threat-service"]

FROM alpine:3.22 AS migrate

RUN addgroup -S app && adduser -S app -G app

COPY --from=build /migrate /usr/local/bin/migrate
COPY migrations /migrations

USER app

ENTRYPOINT ["/usr/local/bin/migrate"]
