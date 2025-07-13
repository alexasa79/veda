package utils

import "fmt"

// Assert panics with a message if the condition is false
func Assert(condition bool, message string) {
	if !condition {
		panic(fmt.Sprintf("assertion failed: %s", message))
	}
}