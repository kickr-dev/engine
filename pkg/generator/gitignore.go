package generator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	// FileGitignore is the filename representation for .gitignore.
	FileGitignore = ".gitignore"
	// GitignoreBaseURL is the Toptal base URL to retrieves .gitignore templates.
	GitignoreBaseURL = "https://www.toptal.com/developers/gitignore/api"
)

// ErrNoClient is returned when input HTTP client is nil.
var ErrNoClient = errors.New("no client provided")

var (
	// ErrInvalidResponse is returned when an HTTP request response status isn't 2XX.
	//
	// When this error is returned, the body is returned alongside it.
	ErrInvalidResponse = errors.New("invalid response from api")

	// ErrNoTemplates is returned when templates slice input in DownloadGitignore function is empty.
	ErrNoTemplates = errors.New("no templates provided")
)

// FetchGitignore fetches a combined .gitignore using the [Gitignore API]
// and returns the obtained result without writing it anywhere.
//
// It's meant for cases where the result must be templatized or combined
// before being written, DownloadGitignore being the shortcut when it can be written as is.
//
//	type config struct { ... }
//
//	func GeneratorGitignore(ctx context.Context, destdir string, c config) error {
//		content, err := generator.FetchGitignore(ctx, cleanhttp.DefaultClient(), "java", "linux")
//		// handle err
//		...
//	}
//
// Note: Full list of templates is available on [Gitignore listing].
//
// [Gitignore API]: https://docs.gitignore.io/use/api
// [Gitignore listing]: https://www.toptal.com/developers/gitignore/api/list
func FetchGitignore(ctx context.Context, httpClient *http.Client, templates ...string) ([]byte, error) {
	if httpClient == nil {
		return nil, ErrNoClient
	}
	if len(templates) == 0 {
		return nil, ErrNoTemplates
	}

	u, err := url.JoinPath(GitignoreBaseURL, strings.Join(templates, ","))
	if err != nil {
		return nil, fmt.Errorf("build gitignore url: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("get '%s': %w", u, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read all: %w", err)
	}
	if response.StatusCode/100 != 2 {
		return nil, fmt.Errorf("invalid response from '%s': %s: %w", u, string(body), ErrInvalidResponse)
	}
	return body, nil
}
