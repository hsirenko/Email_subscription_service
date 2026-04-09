# Build
FROM golang:1.24-alpine AS build
WORKDIR /src

RUN apk add --no-cache ca-certificates git

ENV CGO_ENABLED=0
ENV GOTOOLCHAIN=auto

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# Run
FROM alpine:3.20
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=build /out/api ./api
COPY migrations ./migrations

ENV PORT=8080
EXPOSE 8080

USER nobody:nobody
ENTRYPOINT ["./api"]