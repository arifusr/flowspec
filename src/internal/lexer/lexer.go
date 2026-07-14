package lexer

import (
	"strings"
	"unicode"
)

// Lexer tokenizes FlowSpec DSL source code.
type Lexer struct {
	input   string
	pos     int // current position in input
	readPos int // next read position
	ch      byte
	line    int
	col     int
}

// New creates a new Lexer for the given input.
func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1, col: 0}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
	l.col++
	if l.ch == '\n' {
		l.line++
		l.col = 0
	}
}

func (l *Lexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

// NextToken returns the next token from the input.
func (l *Lexer) NextToken() Token {
	l.skipWhitespace()

	tok := Token{Line: l.line, Column: l.col}

	switch l.ch {
	case 0:
		tok.Type = TOKEN_EOF
		tok.Literal = ""
	case '{':
		tok.Type = TOKEN_LBRACE
		tok.Literal = "{"
	case '}':
		tok.Type = TOKEN_RBRACE
		tok.Literal = "}"
	case '(':
		tok.Type = TOKEN_LPAREN
		tok.Literal = "("
	case ')':
		tok.Type = TOKEN_RPAREN
		tok.Literal = ")"
	case '[':
		tok.Type = TOKEN_LBRACKET
		tok.Literal = "["
	case ']':
		tok.Type = TOKEN_RBRACKET
		tok.Literal = "]"
	case ',':
		tok.Type = TOKEN_COMMA
		tok.Literal = ","
	case '.':
		tok.Type = TOKEN_DOT
		tok.Literal = "."
	case '@':
		tok.Type = TOKEN_AT
		tok.Literal = "@"
	case '#':
		tok.Type = TOKEN_HASH
		tok.Literal = "#"
	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = TOKEN_EQ
			tok.Literal = "=="
		} else {
			tok.Type = TOKEN_ASSIGN
			tok.Literal = "="
		}
	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = TOKEN_NEQ
			tok.Literal = "!="
		} else {
			tok.Type = TOKEN_ILLEGAL
			tok.Literal = string(l.ch)
		}
	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = TOKEN_LTE
			tok.Literal = "<="
		} else {
			tok.Type = TOKEN_LT
			tok.Literal = "<"
		}
	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			tok.Type = TOKEN_GTE
			tok.Literal = ">="
		} else {
			tok.Type = TOKEN_GT
			tok.Literal = ">"
		}
	case '/':
		if l.peekChar() == '/' {
			// Single line comment
			tok.Type = TOKEN_COMMENT
			tok.Literal = l.readLineComment()
			return tok
		} else if l.peekChar() == '*' {
			// Multi-line comment
			tok.Type = TOKEN_COMMENT
			tok.Literal = l.readBlockComment()
			return tok
		} else {
			tok.Type = TOKEN_IDENT
			tok.Literal = "/"
		}
	case '"':
		tok.Type = TOKEN_STRING
		tok.Literal = l.readString('"')
		return tok
	case '\'':
		tok.Type = TOKEN_STRING
		tok.Literal = l.readString('\'')
		return tok
	default:
		if isLetter(l.ch) || l.ch == '_' || l.ch == '$' {
			literal := l.readIdentifier()
			tok.Literal = literal
			tok.Type = LookupIdent(literal)
			// Check if it's a duration or size literal
			if tok.Type == TOKEN_IDENT {
				// check for patterns like 2xx
				if len(literal) == 3 && literal[1] == 'x' && literal[2] == 'x' &&
					literal[0] >= '1' && literal[0] <= '5' {
					tok.Type = TOKEN_IDENT // treat as ident, parser handles status range
				}
			}
			return tok
		} else if isDigit(l.ch) {
			return l.readNumber()
		} else {
			tok.Type = TOKEN_ILLEGAL
			tok.Literal = string(l.ch)
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readIdentifier() string {
	start := l.pos
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' || l.ch == '-' || l.ch == '$' || l.ch == '.' {
		l.readChar()
	}
	return l.input[start:l.pos]
}

// ReadPath reads a path-like token (used after import keyword).
// Includes /, -, ., letters, digits, _
func (l *Lexer) ReadPath() string {
	l.skipWhitespace()
	start := l.pos
	for l.ch != 0 && l.ch != '\n' && l.ch != '\r' && l.ch != ' ' && l.ch != '\t' &&
		l.ch != '{' && l.ch != '}' && l.ch != '(' && l.ch != ')' {
		l.readChar()
	}
	return l.input[start:l.pos]
}

func (l *Lexer) readNumber() Token {
	tok := Token{Line: l.line, Column: l.col}
	start := l.pos
	for isDigit(l.ch) {
		l.readChar()
	}
	// Check for duration/size suffix
	if isLetter(l.ch) {
		suffixStart := l.pos
		for isLetter(l.ch) {
			l.readChar()
		}
		suffix := l.input[suffixStart:l.pos]
		full := l.input[start:l.pos]
		switch strings.ToLower(suffix) {
		case "ms", "s", "m", "h":
			tok.Type = TOKEN_DURATION
			tok.Literal = full
		case "bytes", "kb", "mb", "gb":
			tok.Type = TOKEN_SIZE
			tok.Literal = full
		default:
			tok.Type = TOKEN_IDENT
			tok.Literal = full
		}
		return tok
	}
	tok.Type = TOKEN_INT
	tok.Literal = l.input[start:l.pos]
	return tok
}

func (l *Lexer) readString(quote byte) string {
	l.readChar() // skip opening quote
	var sb strings.Builder
	for l.ch != quote && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case '\\':
				sb.WriteByte('\\')
			case '"':
				sb.WriteByte('"')
			case '\'':
				sb.WriteByte('\'')
			default:
				sb.WriteByte('\\')
				sb.WriteByte(l.ch)
			}
		} else {
			sb.WriteByte(l.ch)
		}
		l.readChar()
	}
	l.readChar() // skip closing quote
	return sb.String()
}

func (l *Lexer) readLineComment() string {
	// skip the //
	l.readChar()
	l.readChar()
	start := l.pos
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
	return strings.TrimSpace(l.input[start:l.pos])
}

func (l *Lexer) readBlockComment() string {
	// skip the /*
	l.readChar()
	l.readChar()
	start := l.pos
	for {
		if l.ch == 0 {
			break
		}
		if l.ch == '*' && l.peekChar() == '/' {
			end := l.pos
			l.readChar() // skip *
			l.readChar() // skip /
			return strings.TrimSpace(l.input[start:end])
		}
		l.readChar()
	}
	return l.input[start:l.pos]
}

func isLetter(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || ch == '_'
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
