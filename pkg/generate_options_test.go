package engine //nolint:testpackage

import (
	"strings"
	"sync"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionsFromContext(t *testing.T) {
	t.Run("success_no_options", func(t *testing.T) {
		// Act
		opts := OptionsFromContext(t.Context())

		// Assert
		assert.Equal(t, Options{}, opts)
	})

	t.Run("success_options", func(t *testing.T) {
		// Arrange
		ctx := ContextWithOptions(t.Context(), Options{
			Force:       true,
			Funcs:       template.FuncMap{"hello": func() string { return "hello" }},
			StripNotice: true,
		})

		// Act
		opts := OptionsFromContext(ctx)

		// Assert
		assert.True(t, opts.Force)
		assert.True(t, opts.StripNotice)
		f, ok := opts.Funcs["hello"].(func() string)
		require.True(t, ok)
		assert.Equal(t, "hello", f())
	})
}

func TestSetLogger(t *testing.T) {
	initial := GetLogger()
	t.Cleanup(func() { SetLogger(initial) })

	t.Run("success_nil_logger", func(t *testing.T) {
		// Act
		SetLogger(nil)

		// Assert
		assert.NotNil(t, GetLogger())
	})

	t.Run("success_logger", func(t *testing.T) {
		// Arrange
		buf := strings.Builder{}

		// Act
		SetLogger(NewTestLogger(&buf))

		// Assert
		GetLogger().Infof("some text to verify")
		assert.Equal(t, "some text to verify", buf.String())
	})

	t.Run("success_concurrent_set_and_get", func(_ *testing.T) {
		// Arrange
		var wg sync.WaitGroup

		// Act
		for range 20 {
			wg.Go(func() { SetLogger(&noopLogger{}) })
			wg.Go(func() { GetLogger() })
		}
		wg.Wait()

		// Assert: no data race (run with -race), no panic
	})
}
