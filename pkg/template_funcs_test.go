package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	engine "github.com/kickr-dev/engine/pkg"
)

func TestMergeMaps(t *testing.T) {
	fm := engine.FuncMap()["map"]
	mergeMap, ok := fm.(func(dest map[string]any, src ...any) (map[string]any, error))
	require.True(t, ok)

	t.Run("error_decode", func(t *testing.T) {
		// Act
		m, err := mergeMap(map[string]any{}, "hey !")

		// Assert
		assert.ErrorContains(t, err, "decode src 0")
		assert.Equal(t, map[string]any{}, m)
	})

	t.Run("error_decode_accumulates_all_sources", func(t *testing.T) {
		// Act
		_, err := mergeMap(map[string]any{}, "hey !", "ho !")

		// Assert
		assert.ErrorContains(t, err, "decode src 0")
		assert.ErrorContains(t, err, "decode src 1")
	})

	t.Run("success", func(t *testing.T) {
		// Act
		m, err := mergeMap(map[string]any{"key": "value"}, map[string]any{"key_one": "value"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"key":     "value",
			"key_one": "value",
		}, m)
	})

	t.Run("success_src_overrides_dst", func(t *testing.T) {
		// Act
		m, err := mergeMap(map[string]any{"key": "original"}, map[string]any{"key": "overridden"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"key": "overridden"}, m)
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

func TestSlug(t *testing.T) {
	fm := engine.FuncMap()["toSlug"]
	toSlug, ok := fm.(func(in string) string)
	require.True(t, ok)

	t.Run("success", func(t *testing.T) {
		// Act
		result := toSlug("something.things_others-and! not/none")

		// Assert
		assert.Equal(t, "something-things-others-and-not-none", result)
	})
}
