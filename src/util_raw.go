//go:build raw

package main

import (
	"C"
	"io"
	"os"
	"sync/atomic"
)

// Main entry point for C programs
//
//export GoMain
func GoMain() {
	main()
}

// Log writer implementation
func NewLogWriter() io.Writer {
	return os.Stderr
}

// Message box implementation using stderr
func ShowInfoDialog(message, title string) {
	atomic.StoreInt32(&errorDialogShown, 1)
	print(title + "\n\n" + message)
}

func ShowErrorDialog(message string) {
	print("I.K.E.M.E.N Error\n\n" + message)
}

// TTF font loading stub
func LoadFntTtf(f *Fnt, fontfile string, filename string, height int32) {
	panic(Error("TrueType fonts are not supported on this platform"))
}
