//go:build windows

package main

func makeStdinHidden() (interface{}, error) {
	return nil, nil
}

func restoreStdin(state interface{}) {}
