package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetString(t *testing.T) {
	var nilStr *string
	result := GetString(nilStr)
	assert.Equal(t, "", result, "expected empty string for nil pointer")

	value := "hello"
	result = GetString(&value)
	assert.Equal(t, "hello", result, "expected value to be returned")
}

func TestGetFloat64(t *testing.T) {
	var nilFloat *float64
	result := GetFloat64(nilFloat)
	assert.Equal(t, 0.0, result, "expected 0.0 for nil pointer")

	f := 3.1415
	result = GetFloat64(&f)
	assert.Equal(t, 3.1415, result, "expected float64 value to be returned")
}

func TestGetBool(t *testing.T) {
	var nilBool *bool
	result := GetBool(nilBool)
	assert.False(t, result, "expected false for nil pointer")

	b := true
	result = GetBool(&b)
	assert.True(t, result, "expected true for pointer to true")

	b = false
	result = GetBool(&b)
	assert.False(t, result, "expected false for pointer to false")
}
