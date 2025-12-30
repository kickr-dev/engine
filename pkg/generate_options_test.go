package engine_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	engine "github.com/kickr-dev/engine/pkg"
)

func TestConfigure(t *testing.T) {
	t.Run("no_options", func(t *testing.T) {
		// Act
		engine.Configure()

		// Assert
		assert.False(t, engine.Forced())
		assert.NotNil(t, engine.GetLogger())
	})

	t.Run("options", func(t *testing.T) {
		// Arrange
		buf := strings.Builder{}

		// Act
		engine.Configure(engine.WithForce(true), engine.WithLogger(engine.NewTestLogger(&buf)))

		// Assert
		assert.True(t, engine.Forced())
		logger := engine.GetLogger()
		require.NotNil(t, logger)
		logger.Infof("some text to verify")
		assert.Equal(t, "some text to verify", buf.String())
	})
}
