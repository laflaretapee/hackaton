FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /api .

FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata && addgroup -S buddy && adduser -S buddy -G buddy
COPY --from=build /api /usr/local/bin/api
USER buddy
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
