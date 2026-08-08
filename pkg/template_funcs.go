package engine

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"text/template"

	"dario.cat/mergo"
	"github.com/go-viper/mapstructure/v2"
	"github.com/goccy/go-yaml"
)

// FuncMap returns a minimal template.FuncMap.
//
// It can be extended with MergeMaps.
func FuncMap() template.FuncMap {
	return template.FuncMap{
		"cutAfter": cutAfter,
		"map":      mergeMaps,
		"toSlug":   ToSlug,
		"toQuery":  toQuery,
		"toYaml":   toYAML,
	}
}

// cutAfter cuts the input string at the first separator appearance
// and returns the resulting string.
func cutAfter(in, sep string) string {
	out, _, _ := strings.Cut(in, sep)
	return out
}

// mergeMaps merges all src maps into dst map,
// returning a joined error if any src isn't a map or fails to merge.
func mergeMaps(dst map[string]any, src ...any) (map[string]any, error) {
	errs := make([]error, 0, len(src))
	for i, in := range src {
		var cast map[string]any
		if err := mapstructure.Decode(in, &cast); err != nil {
			errs = append(errs, fmt.Errorf("decode src %d: %w", i, err))
			continue
		}
		if err := mergo.Merge(&dst, cast, mergo.WithOverride); err != nil {
			errs = append(errs, fmt.Errorf("merge src %d: %w", i, err))
		}
	}
	return dst, errors.Join(errs...)
}

// toQuery transforms a specific into its query parameter format.
func toQuery(in string) string {
	return url.QueryEscape(in)
}

// toYAML takes an interface, marshals it to yaml, and returns a string.
// It will always return a string, even on marshal error (empty string).
//
// This is designed to be called from a go template.
func toYAML(v any) string {
	b, err := yaml.MarshalWithOptions(v, yaml.Indent(2))
	if err != nil {
		// Swallow errors inside of a template.
		return ""
	}
	return string(bytes.TrimSuffix(b, []byte("\n")))
}

var slugRegexp = regexp.MustCompile("[^a-zA-Z0-9]+")

// ToSlug transforms an input string into its slug representation,
// keeping only letters and numbers and replacing everything else by the dash '-'.
func ToSlug(in string) string {
	spaced := strings.TrimSpace(slugRegexp.ReplaceAllString(in, " "))
	return strings.ToLower(strings.ReplaceAll(spaced, " ", "-"))
}
