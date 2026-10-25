FROM golang:1.26.10-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /erp .

FROM alpine:3.23

RUN apk add --no-cache ca-certificates && addgroup -S app && adduser -S -G app app

COPY --from=build /erp /usr/local/bin/erp

USER app
EXPOSE 8080

ENTRYPOINT ["erp"]
