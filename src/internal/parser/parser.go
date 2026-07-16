package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/testing-cli/apitest/internal/ast"
	"github.com/testing-cli/apitest/internal/lexer"
)

// Parser parses FlowSpec DSL tokens into an AST.
type Parser struct {
	l         *lexer.Lexer
	curToken  lexer.Token
	peekToken lexer.Token
	errors    []ParseError
	filename  string
}

// ParseError represents a parsing error with location.
type ParseError struct {
	File    string
	Line    int
	Column  int
	Message string
}

func (e ParseError) Error() string {
	return fmt.Sprintf("[%s:%d:%d] %s", e.File, e.Line, e.Column, e.Message)
}

// New creates a new Parser.
func New(input string, filename string) *Parser {
	l := lexer.New(input)
	p := &Parser{l: l, filename: filename}
	p.nextToken()
	p.nextToken()
	return p
}

// Errors returns all parse errors.
func (p *Parser) Errors() []ParseError {
	return p.errors
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
	// Skip comments
	for p.peekToken.Type == lexer.TOKEN_COMMENT {
		p.peekToken = p.l.NextToken()
	}
}

func (p *Parser) addError(msg string) {
	p.errors = append(p.errors, ParseError{
		File:    p.filename,
		Line:    p.curToken.Line,
		Column:  p.curToken.Column,
		Message: msg,
	})
}

func (p *Parser) expect(t lexer.TokenType) bool {
	if p.peekToken.Type == t {
		p.nextToken()
		return true
	}
	p.addError(fmt.Sprintf("expected token type %d, got %q", t, p.peekToken.Literal))
	return false
}

func (p *Parser) curPos() ast.Position {
	return ast.Position{File: p.filename, Line: p.curToken.Line, Column: p.curToken.Column}
}

// Parse parses the full file and returns an AST File node.
func (p *Parser) Parse() *ast.File {
	file := &ast.File{Position: ast.Position{File: p.filename, Line: 1, Column: 1}}

	for p.curToken.Type != lexer.TOKEN_EOF {
		switch p.curToken.Type {
		case lexer.TOKEN_COMMENT:
			p.nextToken()
		case lexer.TOKEN_AT:
			p.parseAnnotation(file)
		case lexer.TOKEN_PROJECT:
			p.skipProjectBlock()
		case lexer.TOKEN_IMPORT:
			imp := p.parseImport()
			if imp != nil {
				file.Imports = append(file.Imports, *imp)
			}
		case lexer.TOKEN_ENV:
			env := p.parseEnv()
			if env != nil {
				file.Envs = append(file.Envs, *env)
			}
		case lexer.TOKEN_AUTH:
			auth := p.parseAuth()
			if auth != nil {
				file.Auths = append(file.Auths, *auth)
			}
		case lexer.TOKEN_REQUEST:
			req := p.parseRequest(file.Tags)
			if req != nil {
				file.Requests = append(file.Requests, *req)
			}
			file.Tags = nil
		case lexer.TOKEN_FLOW:
			flow := p.parseFlow(file.Tags, file.EnvTag)
			if flow != nil {
				file.Flows = append(file.Flows, *flow)
			}
			file.Tags = nil
			file.EnvTag = ""
		case lexer.TOKEN_FRAGMENT:
			frag := p.parseFragment()
			if frag != nil {
				file.Fragments = append(file.Fragments, *frag)
			}
		default:
			p.addError(fmt.Sprintf("unexpected token %q", p.curToken.Literal))
			p.nextToken()
		}
	}

	return file
}

func (p *Parser) skipProjectBlock() {
	// Skip the entire `project "name" { ... }` block
	p.nextToken() // skip 'project'
	// Skip project name (string)
	if p.curToken.Type == lexer.TOKEN_STRING {
		p.nextToken()
	}
	// Skip the block { ... }
	if p.curToken.Type == lexer.TOKEN_LBRACE {
		depth := 1
		p.nextToken()
		for depth > 0 && p.curToken.Type != lexer.TOKEN_EOF {
			if p.curToken.Type == lexer.TOKEN_LBRACE {
				depth++
			} else if p.curToken.Type == lexer.TOKEN_RBRACE {
				depth--
			}
			p.nextToken()
		}
	}
}

func (p *Parser) parseAnnotation(file *ast.File) {
	p.nextToken() // skip @
	switch p.curToken.Type {
	case lexer.TOKEN_TAGS:
		p.nextToken() // skip 'tags'
		if p.curToken.Type == lexer.TOKEN_LPAREN {
			tags := p.parseTagList()
			file.Tags = append(file.Tags, tags...)
		}
	case lexer.TOKEN_ENV:
		p.nextToken() // skip 'env'
		if p.curToken.Type == lexer.TOKEN_LPAREN {
			p.nextToken() // skip (
			file.EnvTag = p.curToken.Literal
			p.nextToken() // skip env name
			if p.curToken.Type == lexer.TOKEN_RPAREN {
				p.nextToken() // skip )
			}
		}
	case lexer.TOKEN_IMPORT:
		p.nextToken() // skip 'import' after @import (this is a no-op annotation)
	default:
		// Unknown annotation, skip
		p.nextToken()
	}
}

func (p *Parser) parseTagList() []string {
	var tags []string
	p.nextToken() // skip (
	for p.curToken.Type != lexer.TOKEN_RPAREN && p.curToken.Type != lexer.TOKEN_EOF {
		if p.curToken.Type != lexer.TOKEN_COMMA {
			tags = append(tags, p.curToken.Literal)
		}
		p.nextToken()
	}
	if p.curToken.Type == lexer.TOKEN_RPAREN {
		p.nextToken() // skip )
	}
	return tags
}

func (p *Parser) parseImport() *ast.ImportDecl {
	pos := p.curPos()
	p.nextToken() // skip 'import'

	// Import path can contain slashes, so concatenate tokens until we hit a keyword or newline-like boundary
	var pathParts []string
	for p.curToken.Type != lexer.TOKEN_EOF &&
		p.curToken.Type != lexer.TOKEN_IMPORT &&
		p.curToken.Type != lexer.TOKEN_REQUEST &&
		p.curToken.Type != lexer.TOKEN_FLOW &&
		p.curToken.Type != lexer.TOKEN_ENV &&
		p.curToken.Type != lexer.TOKEN_AUTH &&
		p.curToken.Type != lexer.TOKEN_FRAGMENT &&
		p.curToken.Type != lexer.TOKEN_AT {
		if p.curToken.Literal == "/" {
			pathParts = append(pathParts, "/")
		} else {
			pathParts = append(pathParts, p.curToken.Literal)
		}
		p.nextToken()

		// If path is a quoted string, it's just one token
		if len(pathParts) == 1 && (strings.Contains(pathParts[0], "/") || strings.HasSuffix(pathParts[0], ".flow")) {
			break
		}
	}

	path := strings.Join(pathParts, "")
	return &ast.ImportDecl{Position: pos, Path: path}
}

func (p *Parser) parseEnv() *ast.EnvDecl {
	pos := p.curPos()
	p.nextToken() // skip 'env'
	name := p.curToken.Literal
	p.nextToken() // skip name

	env := &ast.EnvDecl{Position: pos, Name: name}
	if p.curToken.Type != lexer.TOKEN_LBRACE {
		p.addError("expected '{' after env name")
		return nil
	}
	p.nextToken() // skip {

	for p.curToken.Type != lexer.TOKEN_RBRACE && p.curToken.Type != lexer.TOKEN_EOF {
		key := p.curToken.Literal
		p.nextToken() // skip key
		if p.curToken.Type == lexer.TOKEN_ASSIGN {
			p.nextToken() // skip =
		}
		a := ast.Assignment{Position: p.curPos(), Key: key}
		// Check for env("VAR") pattern
		if p.curToken.Type == lexer.TOKEN_ENV && p.peekToken.Type == lexer.TOKEN_LPAREN {
			p.nextToken() // skip env
			p.nextToken() // skip (
			a.IsEnvRef = true
			a.EnvVar = p.curToken.Literal
			a.Value = ""
			p.nextToken() // skip var name
			if p.curToken.Type == lexer.TOKEN_RPAREN {
				p.nextToken() // skip )
			}
		} else {
			a.Value = p.curToken.Literal
			p.nextToken()
		}
		env.Assignments = append(env.Assignments, a)
	}
	if p.curToken.Type == lexer.TOKEN_RBRACE {
		p.nextToken() // skip }
	}
	return env
}

func (p *Parser) parseAuth() *ast.AuthDecl {
	pos := p.curPos()
	p.nextToken() // skip 'auth'
	name := p.curToken.Literal
	p.nextToken() // skip name

	auth := &ast.AuthDecl{Position: pos, Name: name}
	if p.curToken.Type != lexer.TOKEN_LBRACE {
		p.addError("expected '{' after auth name")
		return nil
	}
	p.nextToken() // skip {

	for p.curToken.Type != lexer.TOKEN_RBRACE && p.curToken.Type != lexer.TOKEN_EOF {
		switch p.curToken.Type {
		case lexer.TOKEN_HEADER:
			h := p.parseHeader()
			auth.Headers = append(auth.Headers, h)
		case lexer.TOKEN_QUERY:
			q := p.parseQuery()
			auth.Queries = append(auth.Queries, q)
		default:
			p.nextToken()
		}
	}
	if p.curToken.Type == lexer.TOKEN_RBRACE {
		p.nextToken()
	}
	return auth
}

func (p *Parser) parseHeader() ast.HeaderDecl {
	pos := p.curPos()
	p.nextToken() // skip 'header'
	key := p.curToken.Literal
	p.nextToken() // skip key
	if p.curToken.Type == lexer.TOKEN_ASSIGN {
		p.nextToken() // skip =
	}
	value := p.curToken.Literal
	p.nextToken()
	return ast.HeaderDecl{Position: pos, Key: key, Value: value}
}

func (p *Parser) parseQuery() ast.QueryDecl {
	pos := p.curPos()
	p.nextToken() // skip 'query'
	key := p.curToken.Literal
	p.nextToken() // skip key
	if p.curToken.Type == lexer.TOKEN_ASSIGN {
		p.nextToken() // skip =
	}
	value := p.curToken.Literal
	p.nextToken()
	return ast.QueryDecl{Position: pos, Key: key, Value: value}
}

func (p *Parser) parseRequest(tags []string) *ast.RequestDecl {
	pos := p.curPos()
	p.nextToken() // skip 'request'
	name := p.curToken.Literal
	p.nextToken() // skip name

	req := &ast.RequestDecl{Position: pos, Name: name, Tags: tags}

	// Parse parameters: request Name(param1, param2)
	if p.curToken.Type == lexer.TOKEN_LPAREN {
		p.nextToken() // skip (
		for p.curToken.Type != lexer.TOKEN_RPAREN && p.curToken.Type != lexer.TOKEN_EOF {
			if p.curToken.Type != lexer.TOKEN_COMMA {
				req.Params = append(req.Params, p.curToken.Literal)
			}
			p.nextToken()
		}
		if p.curToken.Type == lexer.TOKEN_RPAREN {
			p.nextToken() // skip )
		}
	}

	// Parse extends
	if p.curToken.Type == lexer.TOKEN_EXTENDS {
		p.nextToken() // skip 'extends'
		req.Extends = p.curToken.Literal
		p.nextToken()
	}

	if p.curToken.Type != lexer.TOKEN_LBRACE {
		p.addError("expected '{' after request declaration")
		return nil
	}
	p.nextToken() // skip {

	p.parseRequestBody(req)

	if p.curToken.Type == lexer.TOKEN_RBRACE {
		p.nextToken() // skip }
	}
	return req
}

func (p *Parser) parseRequestBody(req *ast.RequestDecl) {
	for p.curToken.Type != lexer.TOKEN_RBRACE && p.curToken.Type != lexer.TOKEN_EOF {
		switch p.curToken.Type {
		case lexer.TOKEN_GET, lexer.TOKEN_POST, lexer.TOKEN_PUT,
			lexer.TOKEN_PATCH, lexer.TOKEN_DELETE, lexer.TOKEN_HEAD, lexer.TOKEN_OPTIONS:
			req.Method = p.curToken.Literal
			p.nextToken()
			req.URL = p.curToken.Literal
			p.nextToken()
		case lexer.TOKEN_USE:
			p.nextToken() // skip 'use'
			if p.curToken.Type == lexer.TOKEN_AUTH {
				p.nextToken() // skip 'auth'
				req.UseAuth = p.curToken.Literal
				p.nextToken()
			} else {
				p.nextToken() // skip whatever follows use
			}
		case lexer.TOKEN_HEADER:
			h := p.parseHeader()
			req.Headers = append(req.Headers, h)
		case lexer.TOKEN_QUERY:
			q := p.parseQuery()
			req.Queries = append(req.Queries, q)
		case lexer.TOKEN_BODY:
			body := p.parseBodyDecl()
			req.Body = body
		case lexer.TOKEN_EXPECT:
			exp := p.parseExpect()
			if exp != nil {
				req.Expects = append(req.Expects, *exp)
			}
		case lexer.TOKEN_EXTRACT:
			extracts := p.parseExtractBlock()
			req.Extracts = append(req.Extracts, extracts...)
		default:
			p.nextToken()
		}
	}
}

func (p *Parser) parseBodyDecl() *ast.BodyDecl {
	pos := p.curPos()
	p.nextToken() // skip 'body'
	body := &ast.BodyDecl{Position: pos}

	// Check for `body from schema "path"` or `body from file "path"` syntax
	if p.curToken.Type == lexer.TOKEN_FROM {
		p.nextToken() // skip 'from'
		// Accept 'schema', 'file', or just the path directly
		if p.curToken.Literal == "schema" || p.curToken.Literal == "file" {
			p.nextToken() // skip keyword
		}
		body.Type = "schema"
		body.SchemaPath = p.curToken.Literal
		p.nextToken() // skip path string

		// Optional override block { set ... }
		if p.curToken.Type == lexer.TOKEN_LBRACE {
			body.SetOverrides = make(map[string]string)
			p.nextToken() // skip {
			for p.curToken.Type != lexer.TOKEN_RBRACE && p.curToken.Type != lexer.TOKEN_EOF {
				if p.curToken.Type == lexer.TOKEN_SET || p.curToken.Literal == "set" {
					p.nextToken() // skip 'set'
					// Read path (may contain dots and brackets)
					var pathParts []string
					for p.curToken.Type != lexer.TOKEN_ASSIGN &&
						p.curToken.Type != lexer.TOKEN_RBRACE &&
						p.curToken.Type != lexer.TOKEN_EOF {
						pathParts = append(pathParts, p.curToken.Literal)
						p.nextToken()
					}
					path := strings.Join(pathParts, "")
					if p.curToken.Type == lexer.TOKEN_ASSIGN {
						p.nextToken() // skip =
					}
					value := p.curToken.Literal
					p.nextToken() // skip value
					body.SetOverrides[path] = value
				} else {
					p.nextToken()
				}
			}
			if p.curToken.Type == lexer.TOKEN_RBRACE {
				p.nextToken() // skip }
			}
		}
		return body
	}

	body.Type = p.curToken.Literal // json, form, raw, multipart
	p.nextToken()                  // skip type

	if p.curToken.Type != lexer.TOKEN_LBRACE {
		// raw body with content: and content_type:
		body.RawContent = p.curToken.Literal
		p.nextToken()
		return body
	}
	p.nextToken() // skip {

	for p.curToken.Type != lexer.TOKEN_RBRACE && p.curToken.Type != lexer.TOKEN_EOF {
		key := p.curToken.Literal
		p.nextToken() // skip key
		// skip : or =
		if p.curToken.Literal == ":" || p.curToken.Type == lexer.TOKEN_ASSIGN {
			p.nextToken()
		}
		value := p.curToken.Literal
		p.nextToken()
		body.Fields = append(body.Fields, ast.BodyField{Key: key, Value: value})
	}
	if p.curToken.Type == lexer.TOKEN_RBRACE {
		p.nextToken() // skip }
	}
	return body
}

func (p *Parser) parseExpect() *ast.ExpectDecl {
	pos := p.curPos()
	p.nextToken() // skip 'expect'
	exp := &ast.ExpectDecl{Position: pos}

	// Check for 'soft'
	if p.curToken.Type == lexer.TOKEN_SOFT {
		exp.Soft = true
		p.nextToken()
	}

	switch p.curToken.Type {
	case lexer.TOKEN_STATUS:
		exp.Type = "status"
		p.nextToken() // skip 'status'
		// Could be: 200, in [...], 2xx, != 500
		if p.curToken.Type == lexer.TOKEN_NEQ {
			exp.Negated = true
			p.nextToken()
		}
		if p.curToken.Type == lexer.TOKEN_IN {
			p.nextToken() // skip 'in'
			exp.StatusCodes = p.parseIntList()
		} else {
			literal := p.curToken.Literal
			// Check for range like 2xx
			if len(literal) == 3 && literal[1] == 'x' && literal[2] == 'x' {
				exp.StatusRange = literal
			} else {
				code, _ := strconv.Atoi(literal)
				exp.StatusCode = code
			}
			p.nextToken()
		}

	case lexer.TOKEN_JSON:
		exp.Type = "json"
		p.nextToken() // skip 'json'
		exp.JSONPath = p.curToken.Literal
		p.nextToken() // skip jsonpath

		// Parse operator
		switch p.curToken.Type {
		case lexer.TOKEN_EQ:
			exp.Operator = "=="
			p.nextToken()
			exp.Value = p.curToken.Literal
			p.nextToken()
		case lexer.TOKEN_NEQ:
			exp.Operator = "!="
			p.nextToken()
			exp.Value = p.curToken.Literal
			p.nextToken()
		case lexer.TOKEN_GTE:
			exp.Operator = ">="
			p.nextToken()
			exp.Value = p.curToken.Literal
			p.nextToken()
		case lexer.TOKEN_LTE:
			exp.Operator = "<="
			p.nextToken()
			exp.Value = p.curToken.Literal
			p.nextToken()
		case lexer.TOKEN_GT:
			exp.Operator = ">"
			p.nextToken()
			exp.Value = p.curToken.Literal
			p.nextToken()
		case lexer.TOKEN_LT:
			exp.Operator = "<"
			p.nextToken()
			exp.Value = p.curToken.Literal
			p.nextToken()
		case lexer.TOKEN_IS:
			exp.Operator = "is"
			p.nextToken()
			exp.Value = p.curToken.Literal // array, object, number, etc.
			p.nextToken()
		case lexer.TOKEN_EXISTS:
			exp.Operator = "exists"
			p.nextToken()
		case lexer.TOKEN_NOT:
			p.nextToken() // skip 'not'
			exp.Negated = true
			exp.Operator = p.curToken.Literal // exists
			p.nextToken()
		case lexer.TOKEN_LENGTH:
			exp.Operator = "length"
			p.nextToken()
			// Could have >=, <=, etc. before value
			if p.curToken.Type == lexer.TOKEN_GTE || p.curToken.Type == lexer.TOKEN_LTE ||
				p.curToken.Type == lexer.TOKEN_GT || p.curToken.Type == lexer.TOKEN_LT {
				exp.Operator = "length" + p.curToken.Literal
				p.nextToken()
			}
			exp.Value = p.curToken.Literal
			p.nextToken()
		case lexer.TOKEN_MATCHES:
			exp.Operator = "matches"
			p.nextToken()
			exp.Value = p.curToken.Literal
			p.nextToken()
		case lexer.TOKEN_CONTAINS:
			exp.Operator = "contains"
			p.nextToken()
			exp.Value = p.curToken.Literal
			p.nextToken()
		default:
			exp.Operator = p.curToken.Literal
			p.nextToken()
			if p.curToken.Type != lexer.TOKEN_RBRACE && p.curToken.Type != lexer.TOKEN_EOF {
				exp.Value = p.curToken.Literal
				p.nextToken()
			}
		}

	case lexer.TOKEN_HEADER:
		exp.Type = "header"
		p.nextToken() // skip 'header'
		exp.HeaderName = p.curToken.Literal
		p.nextToken()
		// Operator
		switch p.curToken.Type {
		case lexer.TOKEN_EXISTS:
			exp.Operator = "exists"
			p.nextToken()
		case lexer.TOKEN_EQ:
			exp.Operator = "=="
			p.nextToken()
			exp.Value = p.curToken.Literal
			p.nextToken()
		case lexer.TOKEN_CONTAINS:
			exp.Operator = "contains"
			p.nextToken()
			exp.Value = p.curToken.Literal
			p.nextToken()
		case lexer.TOKEN_MATCHES:
			exp.Operator = "matches"
			p.nextToken()
			exp.Value = p.curToken.Literal
			p.nextToken()
		default:
			p.nextToken()
		}

	case lexer.TOKEN_TIME:
		exp.Type = "time"
		p.nextToken() // skip 'time'
		// operator: < or <=
		exp.Operator = p.curToken.Literal
		p.nextToken()
		exp.Duration = p.curToken.Literal
		p.nextToken()

	case lexer.TOKEN_SIZEKEY:
		exp.Type = "size"
		p.nextToken() // skip 'size'
		exp.Operator = p.curToken.Literal
		p.nextToken()
		exp.Size = p.curToken.Literal
		p.nextToken()

	default:
		// Handle 'schema' as expect type (not a keyword token)
		if p.curToken.Literal == "schema" {
			exp.Type = "schema"
			p.nextToken()                  // skip 'schema'
			exp.Value = p.curToken.Literal // schema file path
			p.nextToken()
		} else {
			p.addError(fmt.Sprintf("unknown expect type %q", p.curToken.Literal))
			p.nextToken()
			return nil
		}
	}

	return exp
}

func (p *Parser) parseIntList() []int {
	var codes []int
	if p.curToken.Type == lexer.TOKEN_LBRACKET {
		p.nextToken() // skip [
	}
	for p.curToken.Type != lexer.TOKEN_RBRACKET && p.curToken.Type != lexer.TOKEN_EOF {
		if p.curToken.Type != lexer.TOKEN_COMMA {
			code, _ := strconv.Atoi(p.curToken.Literal)
			codes = append(codes, code)
		}
		p.nextToken()
	}
	if p.curToken.Type == lexer.TOKEN_RBRACKET {
		p.nextToken() // skip ]
	}
	return codes
}

func (p *Parser) parseExtractBlock() []ast.ExtractDecl {
	var extracts []ast.ExtractDecl
	p.nextToken() // skip 'extract'
	if p.curToken.Type != lexer.TOKEN_LBRACE {
		return extracts
	}
	p.nextToken() // skip {

	for p.curToken.Type != lexer.TOKEN_RBRACE && p.curToken.Type != lexer.TOKEN_EOF {
		ext := ast.ExtractDecl{Position: p.curPos()}
		ext.Variable = p.curToken.Literal
		p.nextToken() // skip variable name
		if p.curToken.Type == lexer.TOKEN_FROM {
			p.nextToken() // skip 'from'
		}
		ext.Source = p.curToken.Literal // json, header, cookie
		p.nextToken()
		ext.Path = p.curToken.Literal
		p.nextToken()
		extracts = append(extracts, ext)
	}
	if p.curToken.Type == lexer.TOKEN_RBRACE {
		p.nextToken()
	}
	return extracts
}

func (p *Parser) parseFlow(tags []string, envTag string) *ast.FlowDecl {
	pos := p.curPos()
	p.nextToken() // skip 'flow'
	name := p.curToken.Literal
	p.nextToken() // skip name

	flow := &ast.FlowDecl{Position: pos, Name: name, Tags: tags, EnvTag: envTag}

	if p.curToken.Type != lexer.TOKEN_LBRACE {
		p.addError("expected '{' after flow name")
		return nil
	}
	p.nextToken() // skip {

	for p.curToken.Type != lexer.TOKEN_RBRACE && p.curToken.Type != lexer.TOKEN_EOF {
		switch p.curToken.Type {
		case lexer.TOKEN_DESCRIPTION:
			p.nextToken() // skip 'description'
			flow.Description = p.curToken.Literal
			p.nextToken()
		case lexer.TOKEN_LET:
			l := p.parseLet()
			flow.Lets = append(flow.Lets, l)
		case lexer.TOKEN_STEP:
			step := p.parseStep()
			if step != nil {
				flow.Steps = append(flow.Steps, *step)
			}
		case lexer.TOKEN_TEARDOWN:
			td := p.parseTeardown()
			flow.Teardown = td
		case lexer.TOKEN_INCLUDE:
			p.nextToken() // skip 'include'
			flow.Includes = append(flow.Includes, p.curToken.Literal)
			p.nextToken()
		case lexer.TOKEN_USE:
			// use fragment X
			p.nextToken() // skip 'use'
			if p.curToken.Type == lexer.TOKEN_FRAGMENT {
				p.nextToken() // skip 'fragment'
				// Treat fragment as placeholder step
				step := &ast.StepDecl{
					Position: p.curPos(),
					Name:     "use fragment " + p.curToken.Literal,
				}
				flow.Steps = append(flow.Steps, *step)
				p.nextToken()
			} else {
				p.nextToken()
			}
		case lexer.TOKEN_REPEAT:
			step := p.parseRepeatAsStep()
			if step != nil {
				flow.Steps = append(flow.Steps, *step)
			}
		case lexer.TOKEN_FOR:
			step := p.parseForAsStep()
			if step != nil {
				flow.Steps = append(flow.Steps, *step)
			}
		default:
			p.nextToken()
		}
	}
	if p.curToken.Type == lexer.TOKEN_RBRACE {
		p.nextToken()
	}
	return flow
}

func (p *Parser) parseLet() ast.LetDecl {
	pos := p.curPos()
	p.nextToken() // skip 'let'
	name := p.curToken.Literal
	p.nextToken() // skip name
	if p.curToken.Type == lexer.TOKEN_ASSIGN {
		p.nextToken() // skip =
	}

	// Handle last.json("...") and last.header("...") expressions
	if p.curToken.Literal == "last" || strings.HasPrefix(p.curToken.Literal, "last.") {
		value := p.parseLastExpression()
		return ast.LetDecl{Position: pos, Name: name, Value: value}
	}

	value := p.curToken.Literal
	p.nextToken()
	return ast.LetDecl{Position: pos, Name: name, Value: value}
}

// parseLastExpression parses last.json("$.path") or last.header("Name") or last.status
func (p *Parser) parseLastExpression() string {
	var sb strings.Builder

	// Collect all tokens that form the last.xxx("...") expression
	// Could be: last.json("$.path") — tokenized as: last . json ( "$.path" )
	// Or tokenized as: last.json ( "$.path" ) if lexer reads last.json as one ident
	literal := p.curToken.Literal
	sb.WriteString(literal)
	p.nextToken()

	// If literal already contains the full expression (e.g. last.json), look for (...)
	if strings.HasPrefix(literal, "last.") && !strings.Contains(literal, "(") {
		// Need to read (...) part
		if p.curToken.Type == lexer.TOKEN_LPAREN {
			sb.WriteString("(")
			p.nextToken() // skip (
			// Read everything until )
			depth := 1
			for depth > 0 && p.curToken.Type != lexer.TOKEN_EOF {
				if p.curToken.Type == lexer.TOKEN_LPAREN {
					depth++
				} else if p.curToken.Type == lexer.TOKEN_RPAREN {
					depth--
					if depth == 0 {
						break
					}
				}
				sb.WriteString(p.curToken.Literal)
				p.nextToken()
			}
			sb.WriteString(")")
			if p.curToken.Type == lexer.TOKEN_RPAREN {
				p.nextToken() // skip )
			}
		}
	} else if literal == "last" {
		// last followed by .json(...) or .header(...) or .status
		// Next should be a dot-prefixed identifier
		if strings.HasPrefix(p.curToken.Literal, ".") || p.curToken.Type == lexer.TOKEN_DOT {
			sb.WriteString(p.curToken.Literal)
			p.nextToken()
			// Now should have method name or it was already part of the token
			if p.curToken.Type == lexer.TOKEN_IDENT || p.curToken.Type == lexer.TOKEN_JSON ||
				p.curToken.Type == lexer.TOKEN_HEADER || p.curToken.Type == lexer.TOKEN_STATUS {
				sb.WriteString(p.curToken.Literal)
				p.nextToken()
			}
			// Parse (...) if present
			if p.curToken.Type == lexer.TOKEN_LPAREN {
				sb.WriteString("(")
				p.nextToken()
				depth := 1
				for depth > 0 && p.curToken.Type != lexer.TOKEN_EOF {
					if p.curToken.Type == lexer.TOKEN_LPAREN {
						depth++
					} else if p.curToken.Type == lexer.TOKEN_RPAREN {
						depth--
						if depth == 0 {
							break
						}
					}
					sb.WriteString(p.curToken.Literal)
					p.nextToken()
				}
				sb.WriteString(")")
				if p.curToken.Type == lexer.TOKEN_RPAREN {
					p.nextToken()
				}
			}
		}
	}

	return sb.String()
}

// parseWrite parses `write <source> to "path" [append]`
func (p *Parser) parseWrite() ast.WriteDecl {
	pos := p.curPos()
	p.nextToken() // skip 'write'
	w := ast.WriteDecl{Position: pos}

	// Parse source: last.body, last.json("..."), last.header("..."), last.status, or "{{var}}"
	if p.curToken.Literal == "last" || strings.HasPrefix(p.curToken.Literal, "last.") {
		w.Source = p.parseLastExpression()
	} else {
		// String value or variable interpolation
		w.Source = p.curToken.Literal
		p.nextToken()
	}

	// Expect 'to'
	if p.curToken.Type == lexer.TOKEN_TO {
		p.nextToken() // skip 'to'
	}

	// Path
	w.Path = p.curToken.Literal
	p.nextToken()

	// Optional 'append'
	if p.curToken.Literal == "append" {
		w.Append = true
		p.nextToken()
	}

	return w
}

func (p *Parser) parseStep() *ast.StepDecl {
	pos := p.curPos()
	p.nextToken() // skip 'step'
	name := p.curToken.Literal
	p.nextToken() // skip step name

	step := &ast.StepDecl{Position: pos, Name: name}

	if p.curToken.Type != lexer.TOKEN_LBRACE {
		p.addError("expected '{' after step name")
		return nil
	}
	p.nextToken() // skip {

	for p.curToken.Type != lexer.TOKEN_RBRACE && p.curToken.Type != lexer.TOKEN_EOF {
		switch p.curToken.Type {
		case lexer.TOKEN_WHEN:
			p.nextToken()
			step.When = p.parseConditionExpr()
		case lexer.TOKEN_UNLESS:
			p.nextToken()
			step.Unless = p.parseConditionExpr()
		case lexer.TOKEN_RUN:
			run := p.parseRun()
			step.Run = run
			step.Statements = append(step.Statements, ast.StepStatement{Type: "run", Run: run})
		case lexer.TOKEN_EXPECT:
			exp := p.parseExpect()
			if exp != nil {
				step.Expects = append(step.Expects, *exp)
				step.Statements = append(step.Statements, ast.StepStatement{Type: "expect", Expect: exp})
			}
		case lexer.TOKEN_LET:
			l := p.parseLet()
			step.Lets = append(step.Lets, l)
			step.Statements = append(step.Statements, ast.StepStatement{Type: "let", Let: &l})
		case lexer.TOKEN_WAIT:
			p.nextToken() // skip 'wait'
			step.Wait = p.curToken.Literal
			p.nextToken()
		case lexer.TOKEN_RETRY:
			retry := p.parseRetry()
			step.Retry = retry
		case lexer.TOKEN_LOG:
			p.nextToken() // skip 'log'
			// Expect ( "message" )
			if p.curToken.Type == lexer.TOKEN_LPAREN {
				p.nextToken() // skip (
				msg := p.curToken.Literal
				p.nextToken() // skip message string
				if p.curToken.Type == lexer.TOKEN_RPAREN {
					p.nextToken() // skip )
				}
				step.Logs = append(step.Logs, msg)
				step.Statements = append(step.Statements, ast.StepStatement{Type: "log", Log: msg})
			}
		case lexer.TOKEN_WRITE:
			w := p.parseWrite()
			step.Writes = append(step.Writes, w)
			step.Statements = append(step.Statements, ast.StepStatement{Type: "write", Write: &w})
		default:
			p.nextToken()
		}
	}
	if p.curToken.Type == lexer.TOKEN_RBRACE {
		p.nextToken()
	}
	return step
}

func (p *Parser) parseConditionExpr() string {
	// Simple condition parsing: collect tokens until ; or { or newline-like boundary
	var parts []string
	for p.curToken.Type != lexer.TOKEN_RBRACE &&
		p.curToken.Type != lexer.TOKEN_EOF &&
		p.curToken.Type != lexer.TOKEN_RUN &&
		p.curToken.Type != lexer.TOKEN_EXPECT &&
		p.curToken.Type != lexer.TOKEN_LET &&
		p.curToken.Type != lexer.TOKEN_WAIT &&
		p.curToken.Type != lexer.TOKEN_RETRY {
		if p.curToken.Literal == ";" {
			p.nextToken()
			break
		}
		parts = append(parts, p.curToken.Literal)
		p.nextToken()
	}
	return strings.Join(parts, " ")
}

func (p *Parser) parseRun() *ast.RunDecl {
	pos := p.curPos()
	p.nextToken() // skip 'run'
	run := &ast.RunDecl{Position: pos, Name: p.curToken.Literal}
	p.nextToken() // skip request name

	// Parse arguments: run GetUser(user_id)
	if p.curToken.Type == lexer.TOKEN_LPAREN {
		p.nextToken() // skip (
		for p.curToken.Type != lexer.TOKEN_RPAREN && p.curToken.Type != lexer.TOKEN_EOF {
			if p.curToken.Type != lexer.TOKEN_COMMA {
				run.Args = append(run.Args, p.curToken.Literal)
			}
			p.nextToken()
		}
		if p.curToken.Type == lexer.TOKEN_RPAREN {
			p.nextToken() // skip )
		}
	}

	// Parse override block
	if p.curToken.Type == lexer.TOKEN_LBRACE {
		run.Override = p.parseRequestOverride()
	}

	return run
}

func (p *Parser) parseRequestOverride() *ast.RequestOverride {
	override := &ast.RequestOverride{}
	p.nextToken() // skip {

	for p.curToken.Type != lexer.TOKEN_RBRACE && p.curToken.Type != lexer.TOKEN_EOF {
		switch p.curToken.Type {
		case lexer.TOKEN_BODY:
			override.Body = p.parseBodyDecl()
		case lexer.TOKEN_HEADER:
			h := p.parseHeader()
			override.Headers = append(override.Headers, h)
		case lexer.TOKEN_QUERY:
			q := p.parseQuery()
			override.Queries = append(override.Queries, q)
		case lexer.TOKEN_EXPECT:
			exp := p.parseExpect()
			if exp != nil {
				override.Expects = append(override.Expects, *exp)
			}
		default:
			p.nextToken()
		}
	}
	if p.curToken.Type == lexer.TOKEN_RBRACE {
		p.nextToken()
	}
	return override
}

func (p *Parser) parseRetry() *ast.RetryDecl {
	pos := p.curPos()
	p.nextToken() // skip 'retry'
	retry := &ast.RetryDecl{Position: pos}

	// retry 10 times every 2s until ...
	times, _ := strconv.Atoi(p.curToken.Literal)
	retry.Times = times
	p.nextToken() // skip number

	if p.curToken.Type == lexer.TOKEN_TIMES {
		p.nextToken() // skip 'times'
	}

	if p.curToken.Type == lexer.TOKEN_EVERY {
		p.nextToken() // skip 'every'
		retry.Interval = p.curToken.Literal
		p.nextToken()
	}

	if p.curToken.Type == lexer.TOKEN_UNTIL {
		p.nextToken() // skip 'until'
		// Parse until condition as an expect
		exp := p.parseExpect()
		if exp != nil {
			retry.Condition = *exp
		}
	}

	// Parse the block
	if p.curToken.Type == lexer.TOKEN_LBRACE {
		p.nextToken() // skip {
		for p.curToken.Type != lexer.TOKEN_RBRACE && p.curToken.Type != lexer.TOKEN_EOF {
			if p.curToken.Type == lexer.TOKEN_RUN {
				retry.Run = p.parseRun()
			} else {
				p.nextToken()
			}
		}
		if p.curToken.Type == lexer.TOKEN_RBRACE {
			p.nextToken()
		}
	}

	return retry
}

func (p *Parser) parseTeardown() *ast.TeardownDecl {
	pos := p.curPos()
	p.nextToken() // skip 'teardown'

	td := &ast.TeardownDecl{Position: pos}
	// Optional name
	if p.curToken.Type == lexer.TOKEN_STRING {
		td.Name = p.curToken.Literal
		p.nextToken()
	}

	if p.curToken.Type != lexer.TOKEN_LBRACE {
		return td
	}
	p.nextToken() // skip {

	for p.curToken.Type != lexer.TOKEN_RBRACE && p.curToken.Type != lexer.TOKEN_EOF {
		switch p.curToken.Type {
		case lexer.TOKEN_IGNORE_FAIL:
			td.IgnoreFail = true
			p.nextToken()
		case lexer.TOKEN_WHEN:
			p.nextToken()
			td.When = p.parseConditionExpr()
		case lexer.TOKEN_RUN:
			td.Run = p.parseRun()
		case lexer.TOKEN_STEP:
			step := p.parseStep()
			if step != nil {
				td.Steps = append(td.Steps, *step)
			}
		default:
			p.nextToken()
		}
	}
	if p.curToken.Type == lexer.TOKEN_RBRACE {
		p.nextToken()
	}
	return td
}

func (p *Parser) parseFragment() *ast.FragmentDecl {
	pos := p.curPos()
	p.nextToken() // skip 'fragment'
	name := p.curToken.Literal
	p.nextToken() // skip name

	frag := &ast.FragmentDecl{Position: pos, Name: name}

	if p.curToken.Type != lexer.TOKEN_LBRACE {
		return frag
	}
	p.nextToken() // skip {

	for p.curToken.Type != lexer.TOKEN_RBRACE && p.curToken.Type != lexer.TOKEN_EOF {
		if p.curToken.Type == lexer.TOKEN_STEP {
			step := p.parseStep()
			if step != nil {
				frag.Steps = append(frag.Steps, *step)
			}
		} else {
			p.nextToken()
		}
	}
	if p.curToken.Type == lexer.TOKEN_RBRACE {
		p.nextToken()
	}
	return frag
}

func (p *Parser) parseRepeatAsStep() *ast.StepDecl {
	pos := p.curPos()
	p.nextToken() // skip 'repeat'
	count, _ := strconv.Atoi(p.curToken.Literal)
	p.nextToken() // skip count

	step := &ast.StepDecl{
		Position: pos,
		Name:     fmt.Sprintf("repeat %d", count),
		Repeat:   &ast.RepeatDecl{Position: pos, Count: count},
	}

	if p.curToken.Type == lexer.TOKEN_LBRACE {
		p.nextToken() // skip {
		for p.curToken.Type != lexer.TOKEN_RBRACE && p.curToken.Type != lexer.TOKEN_EOF {
			if p.curToken.Type == lexer.TOKEN_STEP {
				s := p.parseStep()
				if s != nil {
					step.Repeat.Steps = append(step.Repeat.Steps, *s)
				}
			} else if p.curToken.Type == lexer.TOKEN_RUN {
				// Allow direct run in repeat block
				innerStep := &ast.StepDecl{Position: p.curPos(), Run: p.parseRun()}
				step.Repeat.Steps = append(step.Repeat.Steps, *innerStep)
			} else if p.curToken.Type == lexer.TOKEN_LET {
				// wrap let in a step
				innerStep := &ast.StepDecl{Position: p.curPos()}
				innerStep.Lets = append(innerStep.Lets, p.parseLet())
				step.Repeat.Steps = append(step.Repeat.Steps, *innerStep)
			} else if p.curToken.Type == lexer.TOKEN_EXPECT {
				// collect expect
				exp := p.parseExpect()
				if exp != nil && len(step.Repeat.Steps) > 0 {
					last := &step.Repeat.Steps[len(step.Repeat.Steps)-1]
					last.Expects = append(last.Expects, *exp)
				}
			} else {
				p.nextToken()
			}
		}
		if p.curToken.Type == lexer.TOKEN_RBRACE {
			p.nextToken()
		}
	}
	return step
}

func (p *Parser) parseForAsStep() *ast.StepDecl {
	pos := p.curPos()
	p.nextToken() // skip 'for'
	varName := p.curToken.Literal
	p.nextToken() // skip variable name

	if p.curToken.Type == lexer.TOKEN_IN {
		p.nextToken() // skip 'in'
	}

	step := &ast.StepDecl{
		Position: pos,
		Name:     "for " + varName,
		ForLoop:  &ast.ForLoopDecl{Position: pos, Variable: varName},
	}

	// Determine source: csv(...), data(...), or inline [...]
	if p.curToken.Literal == "csv" || p.curToken.Literal == "data" {
		step.ForLoop.Source = p.curToken.Literal
		p.nextToken() // skip csv/data
		if p.curToken.Type == lexer.TOKEN_LPAREN {
			p.nextToken() // skip (
			step.ForLoop.Path = p.curToken.Literal
			p.nextToken() // skip path
			if p.curToken.Type == lexer.TOKEN_RPAREN {
				p.nextToken() // skip )
			}
		}
	} else if p.curToken.Type == lexer.TOKEN_LBRACKET {
		step.ForLoop.Source = "inline"
		// Skip inline array for now (complex parsing)
		depth := 1
		p.nextToken() // skip [
		for depth > 0 && p.curToken.Type != lexer.TOKEN_EOF {
			if p.curToken.Type == lexer.TOKEN_LBRACKET {
				depth++
			} else if p.curToken.Type == lexer.TOKEN_RBRACKET {
				depth--
			}
			if depth > 0 {
				p.nextToken()
			}
		}
		if p.curToken.Type == lexer.TOKEN_RBRACKET {
			p.nextToken()
		}
	}

	// Parse the block
	if p.curToken.Type == lexer.TOKEN_LBRACE {
		p.nextToken() // skip {
		for p.curToken.Type != lexer.TOKEN_RBRACE && p.curToken.Type != lexer.TOKEN_EOF {
			if p.curToken.Type == lexer.TOKEN_STEP {
				s := p.parseStep()
				if s != nil {
					step.ForLoop.Steps = append(step.ForLoop.Steps, *s)
				}
			} else if p.curToken.Type == lexer.TOKEN_LET {
				innerStep := &ast.StepDecl{Position: p.curPos()}
				innerStep.Lets = append(innerStep.Lets, p.parseLet())
				step.ForLoop.Steps = append(step.ForLoop.Steps, *innerStep)
			} else if p.curToken.Type == lexer.TOKEN_RUN {
				innerStep := &ast.StepDecl{Position: p.curPos(), Run: p.parseRun()}
				step.ForLoop.Steps = append(step.ForLoop.Steps, *innerStep)
			} else {
				p.nextToken()
			}
		}
		if p.curToken.Type == lexer.TOKEN_RBRACE {
			p.nextToken()
		}
	}

	return step
}

// Utility to check if a string is likely an identifier used as ident
func isIdent(_ string) bool {
	return true
}
