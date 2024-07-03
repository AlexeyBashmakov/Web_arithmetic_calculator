#!/bin/bash
go build -ldflags="-s -w" ../cmd/main.go

chmod +x *.sh
