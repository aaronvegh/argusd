#!/bin/bash

CURRENT_VERSION=$(cat "VERSION")
NEW_VERSION=$((CURRENT_VERSION+1))

echo -n $NEW_VERSION > ./VERSION

env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-X main.Version=$NEW_VERSION" -a -o argusd-linux-amd64
env CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-X main.Version=$NEW_VERSION" -a -o argusd-darwin-amd64

aws s3 cp argusd-linux-amd64 s3://argusd/argusd/ --acl public-read
aws s3 cp argusd-darwin-amd64 s3://argusd/argusd/ --acl public-read
aws s3 cp VERSION s3://argusd/argusd/ --acl public-read

# rm ./argusd-linux-amd64
# rm ./argusd-darwin-amd64