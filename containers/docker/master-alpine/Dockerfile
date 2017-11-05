FROM alpine:3.5

RUN \
  apk add --update go git make gcc musl-dev linux-headers ca-certificates && \
  git clone --depth 1 --branch release/1.6 https://github.com/amazoncoinio/go-amazoncoin && \
  (cd go-amazoncoin && make geta) && \
  cp go-amazoncoin/build/bin/geta /geta && \
  apk del go git make gcc musl-dev linux-headers && \
  rm -rf /go-amazoncoin && rm -rf /var/cache/apk/*

EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["/geta"]
