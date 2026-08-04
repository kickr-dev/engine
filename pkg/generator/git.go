package generator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/kickr-dev/engine/pkg/files"
)

// FileGitignore is the filename representation for .gitignore.
const FileGitignore = ".gitignore"

var (
	// ErrInvalidResponse is returned when an HTTP request response status isn't 2XX.
	//
	// When this error is returned, the body is returned alongside it.
	ErrInvalidResponse = errors.New("invalid response from api")

	// ErrNoTemplates is returned when templates slice input in DownloadGitignore function is empty.
	ErrNoTemplates = errors.New("no templates provided")
)

// DownloadGitignore downloads a combined .gitginore with the help of https://docs.gitignore.io/use/api
// and writes into input out obtained result.
//
// It can be used as a simple function, calling it directly,
// but can also be used as its expected usage with engine.Generate:
//
//	type config struct { ... }
//
//	func GeneratorGitignore(ctx context.Context, destdir string, c config) error {
//		return generator.DownloadGitignore(ctx, cleanhttp.DefaultClient(), filepath.Join(destdir, generator.FileGitignore), "java", "linux")
//	}
//
// Note: Full list of templates is available here https://www.toptal.com/developers/gitignore/api/list.
func DownloadGitignore(ctx context.Context, httpClient *http.Client, out string, templates ...string) error {
	body, err := FetchGitignore(ctx, httpClient, templates...)
	if err != nil {
		return fmt.Errorf("fetch gitignore: %w", err)
	}

	if err := os.WriteFile(out, body, files.RwRR); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// FetchGitignore fetches a combined .gitignore with the help of https://docs.gitignore.io/use/api
// and returns the obtained result without writing it anywhere.
//
// It's meant for cases where the result must be templatized or combined
// before being written, DownloadGitignore being the shortcut when it can be written as is.
//
// Note: Full list of templates is available here https://www.toptal.com/developers/gitignore/api/list.
func FetchGitignore(ctx context.Context, httpClient *http.Client, templates ...string) ([]byte, error) {
	if httpClient == nil {
		return nil, ErrNoClient
	}
	if len(templates) == 0 {
		return nil, ErrNoTemplates
	}

	// create request
	url := "https://www.toptal.com/developers/gitignore/api/" + strings.Join(templates, ",")
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// fetch .gitignore
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("get '%s': %w", url, err)
	}
	defer response.Body.Close()

	// read and validate body response
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read all: %w", err)
	}
	if response.StatusCode/100 != 2 {
		return nil, fmt.Errorf("invalid response from '%s': %s: %w", url, string(body), ErrInvalidResponse)
	}
	return body, nil
}
