FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/lj-list-api ./cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=build /out/lj-list-api /app/lj-list-api
COPY internal/database/migrations /app/internal/database/migrations

ENV GIN_MODE=release
ENV PORT=10000

EXPOSE 10000

USER nobody

CMD ["./lj-list-api"]
