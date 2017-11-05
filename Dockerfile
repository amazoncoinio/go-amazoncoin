# Build Geth in a stock Go builder container
FROM golang:1.9-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers

ADD . /go-amazoncoin
RUN cd /go-amazoncoin && make geta

# Pull Geth into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go-amazoncoin/build/bin/geta /usr/local/bin/

EXPOSE 8545 8546 30303 30303/udp
ENTRYPOINT ["geta"]
