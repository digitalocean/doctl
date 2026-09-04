//go:build !darwin && !linux && !windows

package deviceid

func read() string { return "" }
