FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ecrm-api ./cmd/api

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata && addgroup -S app && adduser -S app -G app
USER app
COPY --from=builder /out/ecrm-api /usr/local/bin/ecrm-api
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/ecrm-api"]
