#!/bin/sh

# renovate: datasource=golang-version depName=golang
install-tool golang 1.25.0
make testdata
