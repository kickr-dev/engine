package engine

import (
	"bytes"
	"regexp"
)

var noticeLine = regexp.MustCompile(notice)

// noticeBlock matches a whole generated notice line and the blank line following it, if any.
var noticeBlock = regexp.MustCompile(`(?m)^.*` + notice + `.*\n(?:[ \t\r]*\n)?`)

// EmptyPolicy defines the policy to apply when a generated file is empty (see IsEmpty).
//
// By default, the policy is set to PolicyRemoveEmpty.
type EmptyPolicy int

const (
	// PolicyKeepEmpty writes the file even when its generated content is empty.
	PolicyKeepEmpty EmptyPolicy = iota + 1

	// PolicyRemoveEmpty removes the file when its generated content is empty,
	// or doesn't create it when it doesn't exist yet.
	//
	// This is the default behavior.
	PolicyRemoveEmpty
)

// IsEmpty returns true when content counts as empty under policy, which never happens with PolicyKeepEmpty.
//
// Otherwise, content is empty when each of its lines is one of:
//   - a blank line
//   - a generated notice line
//   - a lone comment delimiter ("/*", "*/", "<!--" or "-->")
//   - a shebang line
//
// Since a notice line is ignored as a whole, keep any other comment out of it.
func IsEmpty(content []byte, policy EmptyPolicy) bool {
	if policy == PolicyKeepEmpty {
		return false
	}
	for line := range bytes.Lines(content) {
		if len(bytes.TrimSpace(line)) == 0 || noticeLine.Match(line) {
			continue
		}

		// trim only the regexp \s class, a line with a Unicode space around its delimiter isn't empty
		trimmed := bytes.Trim(line, " \t\n\f\r")
		switch string(trimmed) {
		case "/*", "*/", "<!--", "-->":
		default:
			if !bytes.HasPrefix(trimmed, []byte("#!")) {
				return false
			}
		}
	}
	return true
}
