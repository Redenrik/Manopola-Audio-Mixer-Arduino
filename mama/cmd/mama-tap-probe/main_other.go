//go:build !darwin || !cgo

package main

import "log"

func main() {
	log.Fatal("mama-tap-probe requires macOS with cgo enabled")
}
