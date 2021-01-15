package main

import (
	"errors"
)

type timeout interface {
	Timeout() bool
}

func IsTimeout(err error) bool {
	var t timeout
	if errors.As(err, &t) && t.Timeout() {
		return true
	}
	return false
}
