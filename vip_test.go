package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsVipMatch(t *testing.T) {
	assert.Equal(t, true, isVip("asdf-internal-vip.asdf.com"))
	assert.Equal(t, true, isVip("dev-ssl-vip2.asdf.com"))
	assert.Equal(t, false, isVip("dev-ssl-vipp01.asdf.com"))
	assert.Equal(t, false, isVip("dev-ssl-vipp.asdf.com"))
}
