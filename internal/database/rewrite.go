package database

import "strconv"

// rewritePlaceholders converts SQLite-style `?` positional placeholders into
// PostgreSQL-style `$1, $2, …` placeholders.
//
// It runs a single left-to-right state machine over the query so that `?`
// characters appearing inside string literals ('…'), quoted identifiers ("…"),
// line comments (-- … newline), and block comments (/* … */) are left
// untouched. Only `?` characters in "normal" SQL context are rewritten.
//
// Doubled-quote escapes (” and "") are handled naturally: the closing quote
// flips the state back to normal, and the immediately following quote flips it
// straight back into the quoted state, so the escaped quote never terminates
// the literal for rewriting purposes.
func rewritePlaceholders(query string) string {
	const (
		normal = iota
		inSingleQuote
		inDoubleQuote
		inLineComment
		inBlockComment
	)

	state := normal
	n := 0

	var b []byte // lazily allocated; nil means "no rewrite yet"

	// emit appends the original byte to the output buffer when one exists.
	emit := func(i int) {
		if b != nil {
			b = append(b, query[i])
		}
	}

	for i := 0; i < len(query); i++ {
		c := query[i]

		switch state {
		case normal:
			switch {
			case c == '?':
				n++
				if b == nil {
					b = make([]byte, 0, len(query)+8)
					b = append(b, query[:i]...)
				}
				b = append(b, '$')
				b = append(b, strconv.Itoa(n)...)
				continue
			case c == '\'':
				state = inSingleQuote
			case c == '"':
				state = inDoubleQuote
			case c == '-' && i+1 < len(query) && query[i+1] == '-':
				state = inLineComment
			case c == '/' && i+1 < len(query) && query[i+1] == '*':
				state = inBlockComment
			}
		case inSingleQuote:
			if c == '\'' {
				state = normal
			}
		case inDoubleQuote:
			if c == '"' {
				state = normal
			}
		case inLineComment:
			if c == '\n' {
				state = normal
			}
		case inBlockComment:
			if c == '*' && i+1 < len(query) && query[i+1] == '/' {
				// Emit '*' and '/', consume both, then return to normal.
				emit(i)
				i++
				emit(i)
				state = normal
				continue
			}
		}

		emit(i)
	}

	if b == nil {
		return query
	}
	return string(b)
}
