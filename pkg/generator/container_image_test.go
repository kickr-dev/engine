package generator_test

import (
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kickr-dev/engine/pkg/generator"
)

func TestFetchContainerImage(t *testing.T) {
	ctx := t.Context()

	httpmock.Activate()
	t.Cleanup(httpmock.DeactivateAndReset)

	t.Run("error_no_client", func(t *testing.T) {
		// Act
		_, err := generator.FetchContainerImage(ctx, nil, "docker.io/library/golang", "1.25.3-trixie")

		// Assert
		assert.ErrorIs(t, err, generator.ErrNoClient)
	})

	t.Run("error_invalid_repository", func(t *testing.T) {
		// Arrange
		t.Setenv("DOCKER_CONFIG", t.TempDir())

		// Act
		_, err := generator.FetchContainerImage(ctx, http.DefaultClient, "Invalid Repo", "1.25.3-trixie")

		// Assert
		assert.ErrorContains(t, err, "new repository 'Invalid Repo'")
	})

	t.Run("error_resolve", func(t *testing.T) {
		// Arrange
		t.Setenv("DOCKER_CONFIG", t.TempDir())

		manifest := "https://registry-1.docker.io/v2/library/golang/manifests/1.25.3-trixie"
		httpmock.RegisterResponder(http.MethodHead, manifest, httpmock.NewStringResponder(http.StatusNotFound, ""))

		// Act
		_, err := generator.FetchContainerImage(ctx, http.DefaultClient, "docker.io/library/golang", "1.25.3-trixie")

		// Assert
		assert.ErrorContains(t, err, "resolve 'docker.io/library/golang:1.25.3-trixie'")
	})

	t.Run("success", func(t *testing.T) {
		// Arrange
		t.Setenv("DOCKER_CONFIG", t.TempDir())

		manifest := "https://registry-1.docker.io/v2/library/golang/manifests/1.25.3-trixie"
		digest := "sha256:71d7c66aa2305f7d239c35d3c04b688c709ee0243d792afb868a3731494f08bb"
		httpmock.RegisterResponder(http.MethodHead, manifest,
			httpmock.NewStringResponder(http.StatusOK, "{}").
				HeaderSet(http.Header{"Content-Type": {"application/vnd.oci.image.index.v1+json"}, "Docker-Content-Digest": {digest}}).
				SetContentLength())

		// Act
		image, err := generator.FetchContainerImage(ctx, http.DefaultClient, "docker.io/library/golang", "1.25.3-trixie")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "docker.io/library/golang:1.25.3-trixie@"+digest, image)
	})
}
