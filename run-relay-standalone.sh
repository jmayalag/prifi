#!/usr/bin/env bash
echo "Make sure $GOPATH/src/github.com/dedis/cothority is a git repo, on branch master"
echo "Make sure you called \"cd sda/app; go run prifi.go s\" at least once before running this"
cd sda/app
go run prifi.go r
