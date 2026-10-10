package engine

// Delimiters represents the pair of start and end delimiters for Go template substitution.
type Delimiters struct {
	// EndDelim is the end delimiter of a Go template statement, i.e. >> or }} or ]], etc.
	EndDelim string

	// StartDelim is the start delimiter of a Go template statement, i.e. << or {{ or [[, etc.
	StartDelim string
}

var (
	chevron = Delimiters{
		EndDelim:   ">>",
		StartDelim: "<<",
	}

	bracket = Delimiters{
		EndDelim:   "}}",
		StartDelim: "{{",
	}

	squareBracket = Delimiters{
		EndDelim:   "]]",
		StartDelim: "[[",
	}
)

// DelimitersChevron returns Go template delimiters << and >>.
func DelimitersChevron() Delimiters {
	return chevron
}

// DelimitersBracket returns Go template delimiters {{ and }}.
func DelimitersBracket() Delimiters {
	return bracket
}

// DelimitersSquareBracket returns Go template delimiters [[ and ]].
func DelimitersSquareBracket() Delimiters {
	return squareBracket
}
