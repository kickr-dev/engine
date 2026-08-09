package generator

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

const (
	// CodeOfConductURL is the Contributor Covenant 3.0 markdown source URL.
	CodeOfConductURL = "https://www.contributor-covenant.org/version/3/0/code_of_conduct/code_of_conduct.md"
	// FileCodeOfConduct is the filename representation for CODE_OF_CONDUCT.md.
	FileCodeOfConduct = "CODE_OF_CONDUCT.md"
)

// FetchCodeOfConduct fetches the [Contributor Covenant 3.0] code of conduct markdown
// and returns the obtained result without writing it anywhere.
//
//	type config struct { ... }
//
//	func GeneratorCodeOfConduct(ctx context.Context, destdir string, c config) error {
//		content, err := generator.FetchCodeOfConduct(ctx, cleanhttp.DefaultClient())
//		// handle err
//		...
//	}
//
// Note: the returned content still contains the upstream placeholders
// (e.g. the reporting method note), callers are expected to patch it before writing it out.
//
// [Contributor Covenant 3.0]: https://www.contributor-covenant.org/version/3/0/code_of_conduct
func FetchCodeOfConduct(ctx context.Context, httpClient *http.Client) ([]byte, error) {
	if httpClient == nil {
		return nil, ErrNoClient
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, CodeOfConductURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("get '%s': %w", CodeOfConductURL, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read all: %w", err)
	}
	if response.StatusCode/100 != 2 {
		return nil, fmt.Errorf("invalid response from '%s': %s: %w", CodeOfConductURL, string(body), ErrInvalidResponse)
	}
	return body, nil
}
