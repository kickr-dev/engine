package generator_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/kickr-dev/engine/pkg/generator"
)

func TestFetchGitignore(t *testing.T) {
	ctx := t.Context()

	httpmock.Activate()
	t.Cleanup(httpmock.DeactivateAndReset)

	mockURL := func(t *testing.T, templates ...string) string {
		t.Helper()
		u, err := url.JoinPath(generator.GitignoreBaseURL, strings.Join(templates, ","))
		require.NoError(t, err)
		return u
	}

	t.Run("error_no_client", func(t *testing.T) {
		// Act
		_, err := generator.FetchGitignore(ctx, nil)

		// Assert
		assert.ErrorIs(t, err, generator.ErrNoClient)
	})

	t.Run("error_no_templates", func(t *testing.T) {
		// Act
		_, err := generator.FetchGitignore(ctx, http.DefaultClient)

		// Assert
		assert.ErrorIs(t, err, generator.ErrNoTemplates)
	})

	t.Run("error_http_call", func(t *testing.T) {
		// Arrange
		httpmock.RegisterResponder(http.MethodGet, mockURL(t, "java"),
			httpmock.NewStringResponder(http.StatusInternalServerError, "some error"))

		// Act
		_, err := generator.FetchGitignore(ctx, http.DefaultClient, "java")

		// Assert
		assert.ErrorIs(t, err, generator.ErrInvalidResponse)
		assert.ErrorContains(t, err, "some error")
	})

	t.Run("error_body_too_large", func(t *testing.T) {
		// Arrange
		httpmock.RegisterResponder(http.MethodGet, mockURL(t, "go"),
			httpmock.NewStringResponder(http.StatusOK, strings.Repeat("a", 1<<20+1)))

		// Act
		_, err := generator.FetchGitignore(ctx, http.DefaultClient, "go")

		// Assert
		assert.ErrorIs(t, err, generator.ErrInvalidResponse)
		assert.ErrorContains(t, err, "exceeds")
	})

	t.Run("success", func(t *testing.T) {
		// Arrange
		httpmock.RegisterResponder(http.MethodGet, mockURL(t, "java", "linux"),
			httpmock.NewStringResponder(http.StatusOK, "some content"))

		// Act
		body, err := generator.FetchGitignore(ctx, http.DefaultClient, "java", "linux")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "some content", string(body))
	})
}
