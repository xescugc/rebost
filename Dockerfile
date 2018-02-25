FROM golang:1.9.4 as builder
COPY . /go/src/github.com/xescugc/rebost
WORKDIR /go/src/github.com/xescugc/rebost
RUN go build -o rebost .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o rebost .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github.com/xescugc/rebost/rebost .
