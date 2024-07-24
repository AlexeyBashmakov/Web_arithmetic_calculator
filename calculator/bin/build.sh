#!/bin/bash

go mod tidy

go build -ldflags="-s -w" ../cmd/main.go
