//go:build !windows

package main

func getDesktopWorkArea() (desktopWorkArea, bool) {
	return desktopWorkArea{}, false
}
