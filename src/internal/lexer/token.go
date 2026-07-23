package lexer

// TokenType represents the type of a lexical token.
type TokenType int

const (
	// Special tokens
	TOKEN_ILLEGAL TokenType = iota
	TOKEN_EOF
	TOKEN_COMMENT

	// Literals
	TOKEN_IDENT    // identifier (request names, variable names)
	TOKEN_STRING   // "double quoted" or 'single quoted'
	TOKEN_INT      // 123
	TOKEN_DURATION // 500ms, 2s, 1m
	TOKEN_SIZE     // 100bytes, 1mb

	// Delimiters
	TOKEN_LBRACE   // {
	TOKEN_RBRACE   // }
	TOKEN_LPAREN   // (
	TOKEN_RPAREN   // )
	TOKEN_LBRACKET // [
	TOKEN_RBRACKET // ]
	TOKEN_COMMA    // ,
	TOKEN_DOT      // .
	TOKEN_AT       // @
	TOKEN_HASH     // #

	// Operators
	TOKEN_ASSIGN // =
	TOKEN_EQ     // ==
	TOKEN_NEQ    // !=
	TOKEN_LT     // <
	TOKEN_GT     // >
	TOKEN_LTE    // <=
	TOKEN_GTE    // >=
	TOKEN_PLUS   // +
	TOKEN_MINUS  // -
	TOKEN_STAR   // *
	TOKEN_SLASH  // /

	// Variable reference
	TOKEN_VARREF // {{variable_name}}

	// Keywords
	TOKEN_PROJECT
	TOKEN_ENV
	TOKEN_REQUEST
	TOKEN_FLOW
	TOKEN_FRAGMENT
	TOKEN_AUTH
	TOKEN_IMPORT
	TOKEN_STEP
	TOKEN_RUN
	TOKEN_EXPECT
	TOKEN_EXTRACT
	TOKEN_LET
	TOKEN_HEADER
	TOKEN_BODY
	TOKEN_QUERY
	TOKEN_FROM
	TOKEN_JSON
	TOKEN_FORM
	TOKEN_RAW
	TOKEN_MULTIPART
	TOKEN_STATUS
	TOKEN_TIME
	TOKEN_SIZEKEY
	TOKEN_CONTRACT
	TOKEN_WHEN
	TOKEN_UNLESS
	TOKEN_FOR
	TOKEN_IN
	TOKEN_REPEAT
	TOKEN_WAIT
	TOKEN_RETRY
	TOKEN_TIMES
	TOKEN_EVERY
	TOKEN_UNTIL
	TOKEN_PARALLEL
	TOKEN_INCLUDE
	TOKEN_EXTENDS
	TOKEN_USE
	TOKEN_TEARDOWN
	TOKEN_IGNORE_FAIL
	TOKEN_DESCRIPTION
	TOKEN_TAGS
	TOKEN_BEFORE
	TOKEN_AFTER
	TOKEN_SET
	TOKEN_LOG
	TOKEN_SCRIPT
	TOKEN_NOT
	TOKEN_IS
	TOKEN_EXISTS
	TOKEN_LENGTH
	TOKEN_MATCHES
	TOKEN_CONTAINS
	TOKEN_ARRAY
	TOKEN_OBJECT
	TOKEN_NUMBER
	TOKEN_BOOLEAN
	TOKEN_STRING_KW
	TOKEN_SOFT
	TOKEN_TRUE
	TOKEN_FALSE
	TOKEN_SETTINGS
	TOKEN_VERSION
	TOKEN_DEFAULT_ENV
	TOKEN_TIMEOUT
	TOKEN_FAIL_FAST
	TOKEN_REDACT
	TOKEN_REPORT_DIR
	TOKEN_SPEC
	TOKEN_COOKIE
	TOKEN_WRITE
	TOKEN_TO
	TOKEN_TRANSFORM
	TOKEN_MAP

	// HTTP methods
	TOKEN_GET
	TOKEN_POST
	TOKEN_PUT
	TOKEN_PATCH
	TOKEN_DELETE
	TOKEN_HEAD
	TOKEN_OPTIONS
)

var keywords = map[string]TokenType{
	"project":     TOKEN_PROJECT,
	"env":         TOKEN_ENV,
	"request":     TOKEN_REQUEST,
	"flow":        TOKEN_FLOW,
	"fragment":    TOKEN_FRAGMENT,
	"auth":        TOKEN_AUTH,
	"import":      TOKEN_IMPORT,
	"step":        TOKEN_STEP,
	"run":         TOKEN_RUN,
	"expect":      TOKEN_EXPECT,
	"extract":     TOKEN_EXTRACT,
	"let":         TOKEN_LET,
	"header":      TOKEN_HEADER,
	"body":        TOKEN_BODY,
	"query":       TOKEN_QUERY,
	"from":        TOKEN_FROM,
	"json":        TOKEN_JSON,
	"form":        TOKEN_FORM,
	"raw":         TOKEN_RAW,
	"multipart":   TOKEN_MULTIPART,
	"status":      TOKEN_STATUS,
	"time":        TOKEN_TIME,
	"size":        TOKEN_SIZEKEY,
	"contract":    TOKEN_CONTRACT,
	"when":        TOKEN_WHEN,
	"unless":      TOKEN_UNLESS,
	"for":         TOKEN_FOR,
	"in":          TOKEN_IN,
	"repeat":      TOKEN_REPEAT,
	"wait":        TOKEN_WAIT,
	"retry":       TOKEN_RETRY,
	"times":       TOKEN_TIMES,
	"every":       TOKEN_EVERY,
	"until":       TOKEN_UNTIL,
	"parallel":    TOKEN_PARALLEL,
	"include":     TOKEN_INCLUDE,
	"extends":     TOKEN_EXTENDS,
	"use":         TOKEN_USE,
	"teardown":    TOKEN_TEARDOWN,
	"ignore_fail": TOKEN_IGNORE_FAIL,
	"description": TOKEN_DESCRIPTION,
	"tags":        TOKEN_TAGS,
	"before":      TOKEN_BEFORE,
	"after":       TOKEN_AFTER,
	"set":         TOKEN_SET,
	"log":         TOKEN_LOG,
	"script":      TOKEN_SCRIPT,
	"not":         TOKEN_NOT,
	"is":          TOKEN_IS,
	"exists":      TOKEN_EXISTS,
	"length":      TOKEN_LENGTH,
	"matches":     TOKEN_MATCHES,
	"contains":    TOKEN_CONTAINS,
	"array":       TOKEN_ARRAY,
	"object":      TOKEN_OBJECT,
	"number":      TOKEN_NUMBER,
	"boolean":     TOKEN_BOOLEAN,
	"string":      TOKEN_STRING_KW,
	"soft":        TOKEN_SOFT,
	"true":        TOKEN_TRUE,
	"false":       TOKEN_FALSE,
	"settings":    TOKEN_SETTINGS,
	"version":     TOKEN_VERSION,
	"default_env": TOKEN_DEFAULT_ENV,
	"timeout":     TOKEN_TIMEOUT,
	"fail_fast":   TOKEN_FAIL_FAST,
	"redact":      TOKEN_REDACT,
	"report_dir":  TOKEN_REPORT_DIR,
	"spec":        TOKEN_SPEC,
	"cookie":      TOKEN_COOKIE,
	"write":       TOKEN_WRITE,
	"to":          TOKEN_TO,
	"transform":   TOKEN_TRANSFORM,
	"map":         TOKEN_MAP,
	"GET":         TOKEN_GET,
	"POST":        TOKEN_POST,
	"PUT":         TOKEN_PUT,
	"PATCH":       TOKEN_PATCH,
	"DELETE":      TOKEN_DELETE,
	"HEAD":        TOKEN_HEAD,
	"OPTIONS":     TOKEN_OPTIONS,
}

// LookupIdent returns the token type for the given identifier.
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return TOKEN_IDENT
}

// Token represents a lexical token with position info.
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}
