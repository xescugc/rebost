# build stage
FROM golang:1.14.4 as builder

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

# final stage
FROM alpine

RUN apk --update add ca-certificates

COPY --from=builder /app/rebost /app/

ENTRYPOINT ["/app/rebost"]
