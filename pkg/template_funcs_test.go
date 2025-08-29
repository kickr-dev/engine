package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	engine "github.com/kickr-dev/engine/pkg"
)

func TestMergeMaps(t *testing.T) {
	fm := engine.FuncMap()["map"]
	mergeMap, ok := fm.(func(dest map[string]any, src ...any) map[string]any)
	require.True(t, ok)

	t.Run("error_decode", func(t *testing.T) {
		// Act
		m := mergeMap(map[string]any{}, "hey !")

		// Assert
		assert.Equal(t, map[string]any{"0_decode_error": "'' expected type 'map[string]interface {}', got unconvertible type 'string'"}, m)
	})

	t.Run("success", func(t *testing.T) {
		// Act
		m := mergeMap(map[string]any{"key": "value"}, map[string]any{"key_one": "value"})

		// Assert
		assert.Equal(t, map[string]any{
			"key":     "value",
			"key_one": "value",
		}, m)
	})
}

func TestToQuery(t *testing.T) {
	fm := engine.FuncMap()["toQuery"]
	toQuery, ok := fm.(func(in string) string)
	require.True(t, ok)

	t.Run("success", func(t *testing.T) {
		// Act
		s := toQuery("some string with spaces")

		// Assert
		assert.Equal(t, "some+string+with+spaces", s)
	})
}

func TestToYAML(t *testing.T) {
	fm := engine.FuncMap()["toYaml"]
	toYAML, ok := fm.(func(v any) string)
	require.True(t, ok)

	t.Run("success", func(t *testing.T) {
		// Act
		s := toYAML("{}")

		// Assert
		assert.Equal(t, `"{}"`, s)
	})
}

func TestCutAfter(t *testing.T) {
	fm := engine.FuncMap()["cutAfter"]
	cut, ok := fm.(func(in, sep string) string)
	require.True(t, ok)

	t.Run("success", func(t *testing.T) {
		// Act
		result := cut("something.things", ".")

		// Assert
		assert.Equal(t, "something", result)
	})
}
