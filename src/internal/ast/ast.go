package ast

// Node is the base interface for all AST nodes.
type Node interface {
	Pos() Position
}

// Position represents a location in source code.
type Position struct {
	File   string
	Line   int
	Column int
}

// File represents a parsed .flow file.
type File struct {
	Position  Position
	Imports   []ImportDecl
	Tags      []string
	EnvTag    string
	Envs      []EnvDecl
	Auths     []AuthDecl
	Requests  []RequestDecl
	Flows     []FlowDecl
	Fragments []FragmentDecl
}

func (f *File) Pos() Position { return f.Position }

// ImportDecl represents `import "path/to/file.flow"`
type ImportDecl struct {
	Position Position
	Path     string
}

func (i *ImportDecl) Pos() Position { return i.Position }

// EnvDecl represents `env name { ... }`
type EnvDecl struct {
	Position    Position
	Name        string
	Assignments []Assignment
}

func (e *EnvDecl) Pos() Position { return e.Position }

// Assignment represents `key = "value"` or `key = env("VAR")`
type Assignment struct {
	Position Position
	Key      string
	Value    string
	IsEnvRef bool   // true if value is env("...")
	EnvVar   string // the env var name if IsEnvRef
}

// AuthDecl represents `auth Name { header ... }`
type AuthDecl struct {
	Position Position
	Name     string
	Headers  []HeaderDecl
	Queries  []QueryDecl
}

func (a *AuthDecl) Pos() Position { return a.Position }

// HeaderDecl represents `header Key = "value"`
type HeaderDecl struct {
	Position Position
	Key      string
	Value    string
}

// QueryDecl represents `query key = "value"`
type QueryDecl struct {
	Position Position
	Key      string
	Value    string
}

// RequestDecl represents a `request Name { ... }` block.
type RequestDecl struct {
	Position   Position
	Name       string
	Params     []string // request parameters e.g. request GetUser(user_id)
	Extends    string   // parent request name if extends
	Tags       []string
	UseAuth    string   // `use auth BearerAuth`
	Method     string   // GET, POST, etc.
	URL        string
	Headers    []HeaderDecl
	Queries    []QueryDecl
	Body       *BodyDecl
	Expects    []ExpectDecl
	Extracts   []ExtractDecl
	BeforeHook []HookStatement
	AfterHook  []HookStatement
}

func (r *RequestDecl) Pos() Position { return r.Position }

// BodyDecl represents request body.
type BodyDecl struct {
	Position    Position
	Type        string            // json, form, multipart, raw
	Fields      []BodyField       // for json/form/multipart
	RawContent  string            // for raw body
	ContentType string            // for raw body content type
}

// BodyField is a key-value in body.
type BodyField struct {
	Key   string
	Value string
}

// ExpectDecl represents an `expect` assertion.
type ExpectDecl struct {
	Position Position
	Type     string // status, json, header, time, size, contract
	// Status assertions
	StatusCode  int
	StatusCodes []int
	StatusRange string // "2xx"
	// JSON assertions
	JSONPath  string
	Operator  string // ==, !=, exists, not exists, is, length, matches, >=, <=, >, <, contains
	Value     string
	// Header assertions
	HeaderName string
	// Time/size assertions
	Duration string
	Size     string
	// Negated
	Negated bool
	// Soft assertion
	Soft bool
}

func (e *ExpectDecl) Pos() Position { return e.Position }

// ExtractDecl represents `extract { var from json "$.path" }`
type ExtractDecl struct {
	Position Position
	Variable string
	Source   string // json, header, cookie
	Path     string
}

func (e *ExtractDecl) Pos() Position { return e.Position }

// FlowDecl represents a `flow Name { ... }` block.
type FlowDecl struct {
	Position    Position
	Name        string
	Tags        []string
	EnvTag      string
	Description string
	Lets        []LetDecl
	Steps       []StepDecl
	Teardown    *TeardownDecl
	Includes    []string
}

func (f *FlowDecl) Pos() Position { return f.Position }

// LetDecl represents `let var = "value"`
type LetDecl struct {
	Position Position
	Name     string
	Value    string
}

// StepDecl represents a `step "name" { ... }` block.
type StepDecl struct {
	Position  Position
	Name      string
	When      string // condition for `when`
	Unless    string // condition for `unless`
	Run       *RunDecl
	Expects   []ExpectDecl
	Lets      []LetDecl
	Wait      string // e.g. "3s"
	Retry     *RetryDecl
	Repeat    *RepeatDecl
	ForLoop   *ForLoopDecl
}

func (s *StepDecl) Pos() Position { return s.Position }

// RunDecl represents `run RequestName(args) { overrides }`
type RunDecl struct {
	Position Position
	Name     string
	Args     []string
	Override *RequestOverride
}

// RequestOverride represents inline override block in `run`.
type RequestOverride struct {
	Body    *BodyDecl
	Headers []HeaderDecl
	Expects []ExpectDecl
}

// RetryDecl represents `retry N times every Xs until ... { ... }`
type RetryDecl struct {
	Position  Position
	Times     int
	Interval  string
	Condition ExpectDecl
	Run       *RunDecl
}

// RepeatDecl represents `repeat N { ... }`
type RepeatDecl struct {
	Position Position
	Count    int
	Steps    []StepDecl
}

// ForLoopDecl represents `for row in [...] { ... }` or `for row in csv(...)`
type ForLoopDecl struct {
	Position Position
	Variable string
	Source   string // inline, csv, data
	Path     string // file path for csv/data
	Items    []map[string]string
	Steps    []StepDecl
}

// TeardownDecl represents `teardown { ... }`
type TeardownDecl struct {
	Position   Position
	Name       string
	IgnoreFail bool
	When       string
	Steps      []StepDecl
	Run        *RunDecl
}

// FragmentDecl represents `fragment Name { ... }`
type FragmentDecl struct {
	Position Position
	Name     string
	Steps    []StepDecl
}

func (f *FragmentDecl) Pos() Position { return f.Position }

// HookStatement represents a statement in before/after hooks.
type HookStatement struct {
	Position Position
	Type     string // set, log, script
	Key      string
	Value    string
}
