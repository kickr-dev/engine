package engine //nolint:testpackage

import (
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigure(t *testing.T) {
	t.Run("no_options", func(t *testing.T) {
		// Act
		Configure()

		// Assert
		assert.False(t, Forced())
		assert.NotNil(t, GetLogger())
	})

	t.Run("options", func(t *testing.T) {
		// Arrange
		buf := strings.Builder{}

		// Act
		Configure(
			WithForce(true),
			WithFuncMap(template.FuncMap{"hello": func() string { return "hello" }}),
			WithLogger(NewTestLogger(&buf)))

		// Assert
		assert.True(t, Forced())

		logger := GetLogger()
		require.NotNil(t, logger)
		logger.Infof("some text to verify")
		assert.Equal(t, "some text to verify", buf.String())

		require.NotNil(t, o.funcs)
		f, ok := o.funcs["hello"].(func() string)
		require.True(t, ok)
		assert.Equal(t, "hello", f())
	})
}
