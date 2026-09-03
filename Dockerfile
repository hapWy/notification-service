FROM golang:1.25-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server
RUN CGO_ENABLED=0 go build -o /out/migrate ./cmd/migrate

FROM alpine:3.20
WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=build /out/server ./server
COPY --from=build /out/migrate ./migrate
COPY configs ./configs
COPY migrations ./migrations
COPY docs ./docs

EXPOSE 8080

# Apply migrations, then start the API + worker process.
CMD ["sh", "-c", "./migrate -direction up && ./server"]
