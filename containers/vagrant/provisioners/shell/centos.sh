#!/bin/bash

sudo yum install -y git wget
sudo yum update -y

wget --continue https://storage.googleapis.com/golang/go1.8.1.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.8.1.linux-amd64.tar.gz

GETA_PATH="~vagrant/go/src/github.com/amazoncoinio/go-amazoncoin/build/bin/"

echo "export PATH=$PATH:/usr/local/go/bin:$GETA_PATH" >> ~vagrant/.bashrc
