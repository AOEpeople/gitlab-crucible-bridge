#!/bin/bash

dep ensure

for GOOS in darwin linux windows; do
    for GOARCH in 386 amd64; do
        go build -v -o gitlab-crucible-bridge-$GOOS-$GOARCH
    done
done
