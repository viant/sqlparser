package source

// DollarQuoteDelimiterSize returns the size of an opening $$ or $tag$
// delimiter, or zero for ordinary placeholders such as $name and $1.
func DollarQuoteDelimiterSize[T ~string | ~[]byte](text T) int {
	if len(text) == 0 || text[0] != '$' {
		return 0
	}
	end := 1
	for end < len(text) && identifier(text[end]) {
		end++
	}
	if end < len(text) && text[end] == '$' && (end == 1 || !digit(text[1])) {
		return end + 1
	}
	return 0
}
