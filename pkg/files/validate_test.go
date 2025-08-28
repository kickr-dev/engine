package files_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kickr-dev/engine/pkg/files"
)

func TestValidate(t *testing.T) {
	schema := json.RawMessage(`
{
	"$schema": "https://json-schema.org/draft/2020-12/schema",
	"$id": "sample.schema.testing",
	"title": "Sample schema",
	"description": "Sample schema for testing",
	"type": "object",
	"required": [ "name" ],
	"properties": {
		"name": { "type": "string" }
	}
}`)

	t.Run("error_nil_read_schem", func(t *testing.T) {
		// Act
		err := files.Validate(nil, func(any) error { return nil })

		// Assert
		assert.ErrorIs(t, err, files.ErrNilRead)
	})

	t.Run("error_nil_read_file", func(t *testing.T) {
		// Act
		err := files.Validate(func(any) error { return nil }, nil)

		// Assert
		assert.ErrorIs(t, err, files.ErrNilRead)
	})

	t.Run("error_read_schema", func(t *testing.T) {
		// Act
		err := files.Validate(func(any) error { return errors.New("an error") }, func(any) error { return nil })

		// Assert
		assert.ErrorContains(t, err, "an error")
	})

	t.Run("error_read_file", func(t *testing.T) {
		// Act
		err := files.Validate(func(v any) error { return json.Unmarshal(schema, v) }, func(any) error { return errors.New("an error") })

		// Assert
		assert.ErrorContains(t, err, "an error")
	})

	t.Run("error_compile_schema", func(t *testing.T) {
		// Act
		err := files.Validate(func(any) error { return nil }, func(any) error { return nil })

		// Assert
		assert.ErrorContains(t, err, "compile schema")
	})

	t.Run("error_validation", func(t *testing.T) {
		// Arrange
		readSchema := func(v any) error { return json.Unmarshal(schema, v) }
		readFile := func(v any) error { return json.Unmarshal([]byte(`{}`), v) }

		// Act
		err := files.Validate(readSchema, readFile)

		// Assert
		assert.ErrorContains(t, err, `validate schema:
- at '/': missing property 'name'`)
	})

	t.Run("success_validation", func(t *testing.T) {
		// Arrange
		readSchema := func(v any) error { return json.Unmarshal(schema, v) }
		readFile := func(v any) error { return json.Unmarshal([]byte(`{ "name": "a name" }`), v) }

		// Act
		err := files.Validate(readSchema, readFile)

		// Assert
		assert.NoError(t, err)
	})
}
