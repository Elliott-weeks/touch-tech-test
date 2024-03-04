package models

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type ExampleStruct struct {
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
}

func TestValidateStruct(t *testing.T) {
	payload := ExampleStruct{
		Name:  "", // Name is required
		Email: "invalid-email",
	}

	errors := ValidateStruct(payload)

	assert.Len(t, errors, 2)

	assert.Equal(t, "ExampleStruct.Name", errors[0].Field)
	assert.Equal(t, "required", errors[0].Tag)
	assert.Empty(t, errors[0].Value)

	assert.Equal(t, "ExampleStruct.Email", errors[1].Field)
	assert.Equal(t, "email", errors[1].Tag)
	assert.Empty(t, errors[1].Value)
}
