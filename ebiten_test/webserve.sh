#/bin/bash

env GOOS=js GOARCH=wasm go build -pgo=auto -gcflags '-c 3 -B -wb=false' -o www/ebiten_test.wasm .
go run ./httpd
