//go:build windows && cgo && (ORT || ALL)

package main

// #cgo LDFLAGS: -L${SRCDIR}/resources/lib/windows
import "C"
