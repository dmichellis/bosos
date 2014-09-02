#!/bin/sh

set -ex
go vet 
go test 
go build -v -ldflags "-X main.BuildDate $( date +"%Y%m%d.%H%M%S" ) -X main.GitHash \"$(  git show-ref --head | head -n 1 )\""
