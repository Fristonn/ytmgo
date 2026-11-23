# Build
FROM golang:1.22-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ytmgo .

# Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata \
  && adduser -D -H -u 10001 app
USER app
WORKDIR /app
COPY --from=build /out/ytmgo /app/ytmgo
ENV ADDR=":8080"
EXPOSE 8080
ENTRYPOINT ["/app/ytmgo"]
