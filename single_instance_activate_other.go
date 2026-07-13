//go:build !windows

package main

func grantExistingSingleInstanceForeground(pid int) {}

func activateExistingSingleInstanceWindow(pid int) {}
