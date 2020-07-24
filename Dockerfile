FROM golang:alpine AS builder
WORKDIR /go/src/github.com/petoc/elevation/
RUN apk add upx
COPY main.go go.mod ./
RUN go mod vendor \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -a -o main . \
    && upx --ultra-brute main

FROM alpine:latest
RUN apk update
WORKDIR /opt/elevation/
COPY --from=builder /go/src/github.com/petoc/elevation/main ./elevation
CMD ["./elevation", "-host", "0.0.0.0"]
