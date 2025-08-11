#!/bin/sh

# renovate: datasource=golang-version depName=golang
install-tool golang 1.24.6
make testdata
