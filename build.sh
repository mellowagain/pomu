#!/bin/bash

GIT_HASH=$(git rev-list -1 HEAD)
go build -ldflags "-X main.GitHash=$GIT_HASH"
