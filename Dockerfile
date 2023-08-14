# build stage
FROM golang:1.21 as builder

WORKDIR /app

ARG version

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X github.com/xescugc/rebost/cmd.Version=${version}"

# final stage
FROM alpine

RUN apk --update add ca-certificates

COPY --from=builder /app/rebost /app/

ENTRYPOINT ["/app/rebost"]
