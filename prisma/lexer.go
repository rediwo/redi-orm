package prisma

import (
	"fmt"
)

// TokenType represents the type of token
type TokenType int

const (
	// Special tokens
	ILLEGAL TokenType = iota
	EOF
	WHITESPACE
	COMMENT

	// Identifiers and literals
	IDENT  // model, field names
	STRING // "string literals"
	NUMBER // 123

	// Keywords
	MODEL
	ENUM
	DATASOURCE
	GENERATOR

	// Types
	INT
	STRING_TYPE
	BOOLEAN
	DATETIME
	JSON
	FLOAT
	DECIMAL

	// Operators and delimiters
	LBRACE    // {
	RBRACE    // }
	LBRACKET  // [
	RBRACKET  // ]
	LPAREN    // (
	RPAREN    // )
	AT        // @
	BLOCK_AT  // @@
	QUESTION  // ?
	COMMA     // ,
	EQUALS    // =
	COLON     // :
	DOT       // .

	// Attribute functions
	DEFAULT
	AUTOINCREMENT
	NOW
	UUID
	ENV
	UPDATEDAT
	CUID
	DB
	TRUE
	FALSE
	UNIQUE
	MAP
	INDEX
	ID
	TEXT
	VARCHAR
	MONEY
	JSONB
	UUID_TYPE
	TIMESTAMP
	DATE_TYPE
	TIME_TYPE
	DECIMAL_TYPE
	DOUBLEPRECISION
	REAL
	SMALLINT
	BIGINT_TYPE
	SERIAL
	BIGSERIAL
	CHAR
	INET
	BIT
	VARBIT
	XML
	
	// Referential actions
	CASCADE
	RESTRICT
	NOACTION
	SETNULL
	SETDEFAULT
	ONDELETE
	ONUPDATE
	DBGENERATED
)

// Token represents a token
type Token struct {
	Type     TokenType
	Literal  string
	Line     int
	Column   int
}

// Lexer represents the lexer
type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int
	column       int
}

// keywords maps string literals to their token types
var keywords = map[string]TokenType{
	"model":           MODEL,
	"enum":            ENUM,
	"datasource":      DATASOURCE,
	"generator":       GENERATOR,
	"Int":             INT,
	"String":          STRING_TYPE,
	"Boolean":         BOOLEAN,
	"DateTime":        DATETIME,
	"Json":            JSON,
	"Float":           FLOAT,
	"Decimal":         DECIMAL,
	"default":         DEFAULT,
	"autoincrement":   AUTOINCREMENT,
	"now":             NOW,
	"uuid":            UUID,
	"env":             ENV,
	"updatedAt":       UPDATEDAT,
	"cuid":            CUID,
	"db":              DB,
	"true":            TRUE,
	"false":           FALSE,
	"Text":            TEXT,
	"VarChar":         VARCHAR,
	"Money":           MONEY,
	"JsonB":           JSONB,
	"Uuid":            UUID_TYPE,
	"Timestamp":       TIMESTAMP,
	"Date":            DATE_TYPE,
	"Time":            TIME_TYPE,
	"DoublePrecision": DOUBLEPRECISION,
	"Real":            REAL,
	"SmallInt":        SMALLINT,
	"BigInt":          BIGINT_TYPE,
	"Serial":          SERIAL,
	"BigSerial":       BIGSERIAL,
	"Char":            CHAR,
	"Inet":            INET,
	"Bit":             BIT,
	"VarBit":          VARBIT,
	"Xml":             XML,
	"Cascade":         CASCADE,
	"Restrict":        RESTRICT,
	"NoAction":        NOACTION,
	"SetNull":         SETNULL,
	"SetDefault":      SETDEFAULT,
	"onDelete":        ONDELETE,
	"onUpdate":        ONUPDATE,
	"dbgenerated":     DBGENERATED,
}

// New creates a new lexer instance
func NewLexer(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 0,
	}
	l.readChar()
	return l
}

// readChar reads the next character and advances position in the input
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0 // ASCII NUL character represents EOF
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	
	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
}

// peekChar returns the next character without advancing position
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

// NextToken scans the input and returns the next token
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	tok.Line = l.line
	tok.Column = l.column

	switch l.ch {
	case '{':
		tok = Token{Type: LBRACE, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '}':
		tok = Token{Type: RBRACE, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '[':
		tok = Token{Type: LBRACKET, Literal: string(l.ch), Line: l.line, Column: l.column}
	case ']':
		tok = Token{Type: RBRACKET, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '(':
		tok = Token{Type: LPAREN, Literal: string(l.ch), Line: l.line, Column: l.column}
	case ')':
		tok = Token{Type: RPAREN, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '@':
		if l.peekChar() == '@' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: BLOCK_AT, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else {
			tok = Token{Type: AT, Literal: string(l.ch), Line: l.line, Column: l.column}
		}
	case '?':
		tok = Token{Type: QUESTION, Literal: string(l.ch), Line: l.line, Column: l.column}
	case ',':
		tok = Token{Type: COMMA, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '=':
		tok = Token{Type: EQUALS, Literal: string(l.ch), Line: l.line, Column: l.column}
	case ':':
		tok = Token{Type: COLON, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '.':
		tok = Token{Type: DOT, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '"':
		tok.Type = STRING
		tok.Literal = l.readString()
		tok.Line = l.line
		tok.Column = l.column
	case '/':
		if l.peekChar() == '/' {
			tok.Type = COMMENT
			tok.Literal = l.readComment()
			tok.Line = l.line
			tok.Column = l.column
		} else {
			tok = Token{Type: ILLEGAL, Literal: string(l.ch), Line: l.line, Column: l.column}
		}
	case 0:
		tok.Literal = ""
		tok.Type = EOF
	default:
		if isLetter(l.ch) {
			tok.Line = l.line
			tok.Column = l.column
			tok.Literal = l.readIdentifier()
			tok.Type = lookupIdent(tok.Literal)
			// Don't call l.readChar() here as readIdentifier() already advanced past the identifier
			return tok
		} else if isDigit(l.ch) {
			tok.Line = l.line
			tok.Column = l.column
			tok.Type = NUMBER
			tok.Literal = l.readNumber()
			// Don't call l.readChar() here as readNumber() already advanced past the number
			return tok
		} else {
			tok = Token{Type: ILLEGAL, Literal: string(l.ch), Line: l.line, Column: l.column}
		}
	}

	l.readChar()
	return tok
}

// skipWhitespace skips whitespace characters
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// readIdentifier reads an identifier
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber reads a number
func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readString reads a string literal
func (l *Lexer) readString() string {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
	}
	return l.input[position:l.position]
}

// readComment reads a comment
func (l *Lexer) readComment() string {
	position := l.position
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
	return l.input[position:l.position]
}

// isLetter checks if the character is a letter
func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

// isDigit checks if the character is a digit
func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

// lookupIdent checks if an identifier is a keyword
func lookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

// String returns string representation of TokenType
func (t TokenType) String() string {
	switch t {
	case ILLEGAL:
		return "ILLEGAL"
	case EOF:
		return "EOF"
	case IDENT:
		return "IDENT"
	case STRING:
		return "STRING"
	case NUMBER:
		return "NUMBER"
	case MODEL:
		return "MODEL"
	case ENUM:
		return "ENUM"
	case DATASOURCE:
		return "DATASOURCE"
	case GENERATOR:
		return "GENERATOR"
	case INT:
		return "INT"
	case STRING_TYPE:
		return "STRING_TYPE"
	case BOOLEAN:
		return "BOOLEAN"
	case DATETIME:
		return "DATETIME"
	case JSON:
		return "JSON"
	case FLOAT:
		return "FLOAT"
	case DECIMAL:
		return "DECIMAL"
	case LBRACE:
		return "{"
	case RBRACE:
		return "}"
	case LBRACKET:
		return "["
	case RBRACKET:
		return "]"
	case LPAREN:
		return "("
	case RPAREN:
		return ")"
	case AT:
		return "@"
	case BLOCK_AT:
		return "@@"
	case QUESTION:
		return "?"
	case COMMA:
		return ","
	case EQUALS:
		return "="
	case COLON:
		return ":"
	case DOT:
		return "."
	case DEFAULT:
		return "DEFAULT"
	case AUTOINCREMENT:
		return "AUTOINCREMENT"
	case NOW:
		return "NOW"
	case UUID:
		return "UUID"
	case ENV:
		return "ENV"
	case UPDATEDAT:
		return "UPDATEDAT"
	case CUID:
		return "CUID"
	case DB:
		return "DB"
	case TRUE:
		return "TRUE"
	case FALSE:
		return "FALSE"
	case UNIQUE:
		return "UNIQUE"
	case MAP:
		return "MAP"
	case INDEX:
		return "INDEX"
	case ID:
		return "ID"
	case TEXT:
		return "TEXT"
	case VARCHAR:
		return "VARCHAR"
	case MONEY:
		return "MONEY"
	case JSONB:
		return "JSONB"
	case UUID_TYPE:
		return "UUID_TYPE"
	case TIMESTAMP:
		return "TIMESTAMP"
	case DATE_TYPE:
		return "DATE_TYPE"
	case TIME_TYPE:
		return "TIME_TYPE"
	case DECIMAL_TYPE:
		return "DECIMAL_TYPE"
	case DOUBLEPRECISION:
		return "DOUBLEPRECISION"
	case REAL:
		return "REAL"
	case SMALLINT:
		return "SMALLINT"
	case BIGINT_TYPE:
		return "BIGINT_TYPE"
	case SERIAL:
		return "SERIAL"
	case BIGSERIAL:
		return "BIGSERIAL"
	case CHAR:
		return "CHAR"
	case INET:
		return "INET"
	case BIT:
		return "BIT"
	case VARBIT:
		return "VARBIT"
	case XML:
		return "XML"
	case CASCADE:
		return "CASCADE"
	case RESTRICT:
		return "RESTRICT"
	case NOACTION:
		return "NOACTION"
	case SETNULL:
		return "SETNULL"
	case SETDEFAULT:
		return "SETDEFAULT"
	case ONDELETE:
		return "ONDELETE"
	case ONUPDATE:
		return "ONUPDATE"
	case DBGENERATED:
		return "DBGENERATED"
	default:
		return fmt.Sprintf("TokenType(%d)", int(t))
	}
}

// String returns string representation of Token
func (t Token) String() string {
	return fmt.Sprintf("{Type: %s, Literal: %q, Line: %d, Column: %d}", 
		t.Type.String(), t.Literal, t.Line, t.Column)
}