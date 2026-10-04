package generator

import (
	"context"
	"fmt"
	"net/http"

	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
)

// FetchContainerImage resolves the digest of the given repository and tag from its registry
// and returns the image reference pinned by that digest (e.g. "docker.io/library/golang:1.25.3-trixie@sha256:...").
//
// The repository must include its registry host (e.g. "docker.io/library/golang").
//
// It can be used as a simple function, calling it directly,
// but can also be used as its expected usage with engine.Generate:
//
//	type config struct { ... }
//
//	func GeneratorDockerfile(ctx context.Context, destdir string, c config) error {
//		image, err := generator.FetchContainerImage(ctx, cleanhttp.DefaultClient(), "docker.io/library/golang", "1.25.3-trixie")
//		// handle err
//		...
//	}
//
// Note: registry credentials are read from Docker configuration ($DOCKER_CONFIG or ~/.docker/config.json, with its credentials helpers),
// anonymous access is used when none matches the registry.
func FetchContainerImage(ctx context.Context, httpClient *http.Client, repository, tag string) (string, error) {
	if httpClient == nil {
		return "", ErrNoClient
	}

	store, err := credentials.NewStoreFromDocker(credentials.StoreOptions{DetectDefaultNativeStore: true})
	if err != nil {
		return "", fmt.Errorf("new docker credentials store: %w", err)
	}

	repo, err := remote.NewRepository(repository)
	if err != nil {
		return "", fmt.Errorf("new repository '%s': %w", repository, err)
	}
	repo.Client = &auth.Client{Cache: auth.NewCache(), Client: httpClient, Credential: credentials.Credential(store)}

	desc, err := repo.Resolve(ctx, tag)
	if err != nil {
		return "", fmt.Errorf("resolve '%s:%s': %w", repository, tag, err)
	}
	return fmt.Sprintf("%s:%s@%s", repository, tag, desc.Digest), nil
}
