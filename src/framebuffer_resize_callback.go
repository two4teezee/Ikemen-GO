//go:build !kinc

package main

import glfw "github.com/go-gl/glfw/v3.3/glfw"

func framebufferSizeCallback(w *glfw.Window, width int, height int) {
	sys.setWindowSize(int32(width), int32(height))
	if gfx != nil {
		gfx.Resize(int32(width), int32(height))
	}
}
