//go:build linux && cgo && (ORT || ALL)

package main

// #cgo LDFLAGS: -L${SRCDIR}/resources/lib/linux
import "C"
