package sqlguard

import "strings"

// readToken preserves quoted identifiers and literal boundaries. The protected
// grammar deliberately rejects dialect-dependent escapes and executable comments
// rather than guessing the server's SQL mode.
type readToken struct {
	text string
	kind byte // w: word, i: quoted identifier, v: value, p: punctuation
}

func lexRead(query string) ([]readToken, bool) {
	var tokens []readToken
	for i := 0; i < len(query); {
		c := query[i]
		if strings.ContainsRune(" \t\r\n\f", rune(c)) {
			i++
			continue
		}
		if c == '-' && i+1 < len(query) && query[i+1] == '-' {
			// MySQL requires whitespace after --; do not erase ambiguous input.
			if i+2 < len(query) && !strings.ContainsRune(" \t\r\n", rune(query[i+2])) {
				return nil, false
			}
			for i < len(query) && query[i] != '\n' && query[i] != '\r' {
				i++
			}
			continue
		}
		if c == '/' && i+1 < len(query) && query[i+1] == '*' {
			i += 2
			end := strings.Index(query[i:], "*/")
			if end < 0 {
				return nil, false
			}
			body := query[i : i+end]
			if strings.HasPrefix(body, "!") || strings.HasPrefix(body, "+") || strings.HasPrefix(strings.ToUpper(body), "M!") || strings.Contains(body, "/*") {
				return nil, false
			}
			i += end + 2
			continue
		}
		if c == '\'' || c == '"' || c == '`' {
			start := i
			i++
			closed := false
			for i < len(query) {
				if query[i] == '\\' {
					return nil, false
				}
				if query[i] == c {
					i++
					if i < len(query) && query[i] == c {
						i++
						continue
					}
					closed = true
					break
				}
				i++
			}
			if !closed {
				return nil, false
			}
			kind := byte('i')
			if c == '\'' {
				kind = 'v'
			}
			tokens = append(tokens, readToken{query[start:i], kind})
			continue
		}
		if c == '?' || c == '$' {
			start := i
			i++
			if c == '$' {
				for i < len(query) && query[i] >= '0' && query[i] <= '9' {
					i++
				}
				if i == start+1 {
					return nil, false
				}
			}
			tokens = append(tokens, readToken{query[start:i], 'v'})
			continue
		}
		if c >= '0' && c <= '9' {
			start := i
			for i < len(query) && query[i] >= '0' && query[i] <= '9' {
				i++
			}
			if i+1 < len(query) && query[i] == '.' && query[i+1] >= '0' && query[i+1] <= '9' {
				i++
				for i < len(query) && query[i] >= '0' && query[i] <= '9' {
					i++
				}
			}
			tokens = append(tokens, readToken{query[start:i], 'v'})
			continue
		}
		if asciiWord(c) {
			start := i
			i++
			for i < len(query) && (asciiWord(query[i]) || query[i] >= '0' && query[i] <= '9') {
				i++
			}
			tokens = append(tokens, readToken{strings.ToUpper(query[start:i]), 'w'})
			continue
		}
		if strings.ContainsRune("(),.;*+-/%=<>!", rune(c)) {
			start := i
			i++
			if i < len(query) && strings.ContainsRune("<>!", rune(c)) && query[i] == '=' {
				i++
			} else if c == '<' && i < len(query) && query[i] == '>' {
				i++
			}
			tokens = append(tokens, readToken{query[start:i], 'p'})
			continue
		}
		return nil, false
	}
	return tokens, true
}

func asciiWord(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' }

// readParser recognizes a bounded, portable read grammar. Unsupported syntax is
// an approval requirement, not a claim that the statement actually writes.
// Routine calls are limited to standard aggregates; qualified/quoted routines,
// table functions, casts and dialect extensions require approval. Database roles
// and trusted schema objects remain part of the execution boundary.
type readParser struct {
	tokens     []readToken
	pos, depth int
}

func understoodRead(query string) bool {
	tokens, ok := lexRead(query)
	if !ok || len(tokens) == 0 {
		return false
	}
	p := readParser{tokens: tokens}
	for p.pos < len(tokens) {
		if !p.statement() {
			return false
		}
		if p.pos == len(tokens) {
			return true
		}
		if !p.take(";") {
			return false
		}
	}
	return true
}

func (p *readParser) peek() string {
	if p.pos == len(p.tokens) {
		return ""
	}
	return p.tokens[p.pos].text
}
func (p *readParser) take(s string) bool {
	if p.peek() != s {
		return false
	}
	p.pos++
	return true
}
func (p *readParser) identifier() bool {
	if p.pos == len(p.tokens) {
		return false
	}
	t := p.tokens[p.pos]
	// Oracle sequence pseudocolumns advance state without call parentheses.
	if strings.EqualFold(strings.Trim(t.text, "\"`"), "NEXTVAL") {
		return false
	}
	if t.kind != 'i' && (t.kind != 'w' || reservedReadWord(t.text)) {
		return false
	}
	p.pos++
	return true
}
func reservedReadWord(s string) bool {
	if mutatingKeywords[s] {
		return true
	}
	switch s {
	case "SELECT", "FROM", "WHERE", "JOIN", "INNER", "LEFT", "RIGHT", "FULL", "OUTER", "CROSS", "ON", "USING", "AS", "WITH", "RECURSIVE", "DISTINCT", "ALL", "UNION", "INTERSECT", "EXCEPT", "GROUP", "BY", "HAVING", "ORDER", "ASC", "DESC", "NULLS", "FIRST", "LAST", "LIMIT", "OFFSET", "FETCH", "FOR", "INTO", "OUTFILE", "DUMPFILE", "SETTINGS", "FORMAT", "PROCEDURE", "SET", "USE", "PRAGMA", "TABLE", "VALUES", "EXPLAIN", "ANALYZE", "AND", "OR", "NOT", "IS", "IN", "LIKE", "BETWEEN", "EXISTS", "NULL", "TRUE", "FALSE", "CASE", "WHEN", "THEN", "ELSE", "END":
		return true
	default:
		return false
	}
}
func (p *readParser) name() bool {
	if !p.identifier() {
		return false
	}
	for p.take(".") {
		if !p.identifier() {
			return false
		}
	}
	return true
}
func (p *readParser) statement() bool {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > 64 {
		return false
	}
	if p.take("EXPLAIN") {
		if !p.take("ANALYZE") {
			p.take("PIPELINE")
		}
		return p.query()
	}
	if p.take("SHOW") {
		// Only standalone metadata forms; never SHOW ... with arbitrary suffixes.
		return p.take("TABLES") || p.take("DATABASES") || p.take("SCHEMAS")
	}
	if p.take("DESCRIBE") || p.take("DESC") {
		return p.name()
	}
	if p.take("PRAGMA") {
		// These SQLite introspection forms accept a name, never configuration.
		if !p.take("TABLE_INFO") && !p.take("TABLE_XINFO") && !p.take("INDEX_INFO") && !p.take("INDEX_LIST") && !p.take("FOREIGN_KEY_LIST") {
			return false
		}
		if !p.take("(") {
			return false
		}
		if !p.identifier() {
			if p.pos == len(p.tokens) || p.tokens[p.pos].kind != 'v' {
				return false
			}
			p.pos++
		}
		return p.take(")")
	}
	return p.query()
}
func (p *readParser) query() bool {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > 64 {
		return false
	}
	if p.take("WITH") {
		p.take("RECURSIVE")
		for {
			if !p.identifier() {
				return false
			}
			if p.take("(") {
				if !p.identifier() {
					return false
				}
				for p.take(",") {
					if !p.identifier() {
						return false
					}
				}
				if !p.take(")") {
					return false
				}
			}
			if !p.take("AS") || !p.take("(") || !p.query() || !p.take(")") {
				return false
			}
			if !p.take(",") {
				break
			}
		}
	}
	if !p.selectBody() {
		return false
	}
	for p.take("UNION") || p.take("INTERSECT") || p.take("EXCEPT") {
		if !p.take("ALL") {
			p.take("DISTINCT")
		}
		if !p.selectBody() {
			return false
		}
	}
	if p.take("ORDER") {
		if !p.take("BY") {
			return false
		}
		for {
			if !p.expression() {
				return false
			}
			if !p.take("ASC") {
				p.take("DESC")
			}
			if p.take("NULLS") && !p.take("FIRST") && !p.take("LAST") {
				return false
			}
			if !p.take(",") {
				break
			}
		}
	}
	if p.take("LIMIT") {
		if !p.value() {
			return false
		}
		if p.take(",") && !p.value() {
			return false
		}
	}
	if p.take("OFFSET") && !p.value() {
		return false
	}
	return true
}
func (p *readParser) selectBody() bool {
	if p.take("TABLE") {
		return p.name()
	}
	if p.take("VALUES") {
		for {
			if !p.take("(") || !p.expressions() || !p.take(")") {
				return false
			}
			if !p.take(",") {
				return true
			}
		}
	}
	if !p.take("SELECT") {
		return false
	}
	if !p.take("DISTINCT") {
		p.take("ALL")
	}
	for {
		if !p.expression() {
			return false
		}
		if p.take("AS") && !p.identifier() {
			return false
		}
		if !p.take(",") {
			break
		}
	}
	if p.take("FROM") {
		if !p.relation() {
			return false
		}
		for {
			if p.take(",") {
				if !p.relation() {
					return false
				}
				continue
			}
			qualified := false
			if p.take("LEFT") || p.take("RIGHT") || p.take("FULL") {
				p.take("OUTER")
				qualified = true
			} else if p.take("INNER") || p.take("CROSS") {
				qualified = true
			}
			if !p.take("JOIN") {
				if qualified {
					return false
				}
				break
			}
			if !p.relation() {
				return false
			}
			if p.take("ON") {
				if !p.expression() {
					return false
				}
			} else if p.take("USING") {
				if !p.take("(") || !p.identifier() {
					return false
				}
				for p.take(",") {
					if !p.identifier() {
						return false
					}
				}
				if !p.take(")") {
					return false
				}
			}
		}
	}
	if p.take("WHERE") && !p.expression() {
		return false
	}
	if p.take("GROUP") && (!p.take("BY") || !p.expressions()) {
		return false
	}
	if p.take("HAVING") && !p.expression() {
		return false
	}
	return true
}
func (p *readParser) relation() bool {
	if p.take("(") {
		if !p.query() || !p.take(")") {
			return false
		}
	} else if !p.name() {
		return false
	}
	if p.take("AS") {
		return p.identifier()
	}
	return true
}
func (p *readParser) expressions() bool {
	if !p.expression() {
		return false
	}
	for p.take(",") {
		if !p.expression() {
			return false
		}
	}
	return true
}
func (p *readParser) value() bool {
	if p.pos == len(p.tokens) || p.tokens[p.pos].kind != 'v' {
		return false
	}
	p.pos++
	return true
}
func (p *readParser) expression() bool {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > 64 || !p.atom() {
		return false
	}
	for {
		switch p.peek() {
		case "=", "<>", "!=", "<", ">", "<=", ">=", "+", "-", "*", "/", "%", "AND", "OR", "LIKE":
			p.pos++
			if !p.atom() {
				return false
			}
		case "IS":
			p.pos++
			p.take("NOT")
			if !p.take("NULL") && !p.take("TRUE") && !p.take("FALSE") {
				return false
			}
		case "IN":
			p.pos++
			if !p.take("(") {
				return false
			}
			if p.peek() == "SELECT" || p.peek() == "WITH" {
				if !p.query() {
					return false
				}
			} else if !p.expressions() {
				return false
			}
			if !p.take(")") {
				return false
			}
		case "BETWEEN":
			p.pos++
			if !p.atom() || !p.take("AND") || !p.atom() {
				return false
			}
		default:
			return true
		}
	}
}
func (p *readParser) atom() bool {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > 64 {
		return false
	}
	if p.take("NOT") || p.take("+") || p.take("-") {
		return p.atom()
	}
	if p.take("EXISTS") {
		return p.take("(") && p.query() && p.take(")")
	}
	if p.take("(") {
		if p.peek() == "SELECT" || p.peek() == "WITH" {
			if !p.query() {
				return false
			}
		} else if !p.expression() {
			return false
		}
		return p.take(")")
	}
	if p.value() || p.take("NULL") || p.take("TRUE") || p.take("FALSE") || p.take("*") {
		return true
	}
	// Only unqualified standard aggregates are in the portable subset.
	if p.pos+1 < len(p.tokens) && p.tokens[p.pos].kind == 'w' && p.tokens[p.pos+1].text == "(" {
		switch p.peek() {
		case "COUNT", "SUM", "AVG", "MIN", "MAX":
		default:
			return false
		}
		p.pos += 2
		p.take("DISTINCT")
		return p.expression() && p.take(")")
	}
	if !p.identifier() {
		return false
	}
	for p.take(".") {
		if p.take("*") {
			return true
		}
		if !p.identifier() {
			return false
		}
	}
	return true
}
