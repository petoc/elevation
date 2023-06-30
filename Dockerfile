FROM golang:alpine AS builder
WORKDIR /go/src/github.com/petoc/elevation/
COPY *.go go.mod go.sum ./
RUN go mod tidy \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -a -o elevation .
# RUN apk add upx && upx --ultra-brute elevation

FROM alpine:edge
RUN apk update
WORKDIR /opt/elevation/
COPY --from=builder /go/src/github.com/petoc/elevation/elevation ./elevation
CMD ["./elevation", "-host", "0.0.0.0"]
