.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/amazoncoinio/go-amazoncoin/cmd/swarm
	go build -o bin/geta     github.com/amazoncoinio/go-amazoncoin/cmd/geta
	go build -o bin/bootnode github.com/amazoncoinio/go-amazoncoin/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/amazoncoinio/go-amazoncoin/swarm/...
