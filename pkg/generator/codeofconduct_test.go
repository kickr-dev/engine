package generator_test

import (
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kickr-dev/engine/pkg/generator"
)

func TestFetchCodeOfConduct(t *testing.T) {
	ctx := t.Context()

	httpmock.Activate()
	t.Cleanup(httpmock.DeactivateAndReset)

	t.Run("error_no_client", func(t *testing.T) {
		// Act
		_, err := generator.FetchCodeOfConduct(ctx, nil)

		// Assert
		assert.ErrorIs(t, err, generator.ErrNoClient)
	})

	t.Run("error_http_call", func(t *testing.T) {
		// Arrange
		httpmock.RegisterResponder(http.MethodGet, generator.CodeOfConductURL,
			httpmock.NewStringResponder(http.StatusInternalServerError, "some error"))

		// Act
		_, err := generator.FetchCodeOfConduct(ctx, http.DefaultClient)

		// Assert
		assert.ErrorIs(t, err, generator.ErrInvalidResponse)
		assert.ErrorContains(t, err, "some error")
	})

	t.Run("success", func(t *testing.T) {
		// Arrange
		httpmock.RegisterResponder(http.MethodGet, generator.CodeOfConductURL,
			httpmock.NewStringResponder(http.StatusOK, "some content"))

		// Act
		body, err := generator.FetchCodeOfConduct(ctx, http.DefaultClient)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "some content", string(body))
	})
}
