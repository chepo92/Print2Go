//go:build windows
// +build windows

package camera

// RunPipe es un stub en Windows: la funcionalidad de webcam no está soportada.
func RunPipe(device string, width, height int) {
	println("Webcam no soportada en Windows")
}
