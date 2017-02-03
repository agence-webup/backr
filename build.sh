#! /bin/bash

cd cmd/backr
gox -osarch="linux/arm linux/amd64" -output="backr_{{.OS}}_{{.Arch}}"