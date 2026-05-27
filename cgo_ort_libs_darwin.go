//go:build darwin && cgo && (ORT || ALL)

package main

// #cgo LDFLAGS: -L${SRCDIR}/resources/lib/darwin
import "C"
