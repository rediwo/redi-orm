package sql

import (
	"fmt"
	"strings"
)

// TokenType represents the type of token
type TokenType int

const (
	// Special tokens
	TokenIllegal TokenType = iota
	TokenEOF

	// Literals
	TokenIdent  // table names, column names
	TokenInt    // 123
	TokenFloat  // 123.45
	TokenString // 'string' or "string"

	// Keywords
	TokenSelect
	TokenFrom
	TokenWhere
	TokenInsert
	TokenInto
	TokenValues
	TokenUpdate
	TokenSet
	TokenDelete
	TokenOrder
	TokenBy
	TokenGroup
	TokenHaving
	TokenLimit
	TokenOffset
	TokenJoin
	TokenInner
	TokenLeft
	TokenRight
	TokenFull
	TokenOn
	TokenAs
	TokenAnd
	TokenOr
	TokenNot
	TokenIn
	TokenIs
	TokenNull
	TokenLike
	TokenBetween
	TokenDistinct
	TokenTrue
	TokenFalse

	// Operators
	TokenEqual        // =
	TokenNotEqual     // !=, <>
	TokenLess         // <
	TokenLessEqual    // <=
	TokenGreater      // >
	TokenGreaterEqual // >=

	// Delimiters
	TokenComma     // ,
	TokenSemicolon // ;
	TokenLParen    // (
	TokenRParen    // )
	TokenStar      // *
	TokenQuestion  // ?
	TokenDot       // .
)

// Token represents a lexical token
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

// Lexer represents the lexical analyzer
type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int  // current line number
	column       int  // current column number
}

// Keywords map
var keywords = map[string]TokenType{
	"SELECT":   TokenSelect,
	"FROM":     TokenFrom,
	"WHERE":    TokenWhere,
	"INSERT":   TokenInsert,
	"INTO":     TokenInto,
	"VALUES":   TokenValues,
	"UPDATE":   TokenUpdate,
	"SET":      TokenSet,
	"DELETE":   TokenDelete,
	"ORDER":    TokenOrder,
	"BY":       TokenBy,
	"GROUP":    TokenGroup,
	"HAVING":   TokenHaving,
	"LIMIT":    TokenLimit,
	"OFFSET":   TokenOffset,
	"JOIN":     TokenJoin,
	"INNER":    TokenInner,
	"LEFT":     TokenLeft,
	"RIGHT":    TokenRight,
	"FULL":     TokenFull,
	"ON":       TokenOn,
	"AS":       TokenAs,
	"AND":      TokenAnd,
	"OR":       TokenOr,
	"NOT":      TokenNot,
	"IN":       TokenIn,
	"IS":       TokenIs,
	"NULL":     TokenNull,
	"LIKE":     TokenLike,
	"BETWEEN":  TokenBetween,
	"DISTINCT": TokenDistinct,
	"TRUE":     TokenTrue,
	"FALSE":    TokenFalse,
}

// NewLexer creates a new lexer instance
func NewLexer(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 0,
	}
	l.readChar()
	return l
}

// readChar reads the next character and advances position
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0 // ASCII NUL character represents "EOF"
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

// skipWhitespace skips whitespace characters
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// readIdentifier reads an identifier or keyword
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber reads a number (integer or float)
func (l *Lexer) readNumber() (string, TokenType) {
	position := l.position
	tokenType := TokenInt

	for isDigit(l.ch) {
		l.readChar()
	}

	// Check for decimal point
	if l.ch == '.' && isDigit(l.peekChar()) {
		tokenType = TokenFloat
		l.readChar() // consume '.'
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	return l.input[position:l.position], tokenType
}

// readString reads a quoted string
func (l *Lexer) readString(delimiter byte) string {
	position := l.position + 1 // skip opening quote
	for {
		l.readChar()
		if l.ch == delimiter || l.ch == 0 {
			break
		}
		// Handle escaped quotes
		if l.ch == '\\' {
			l.readChar()
		}
	}
	return l.input[position:l.position]
}

// NextToken returns the next token
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	tok.Line = l.line
	tok.Column = l.column

	switch l.ch {
	case '=':
		tok = Token{Type: TokenEqual, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenNotEqual, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else {
			tok = Token{Type: TokenIllegal, Literal: string(l.ch), Line: l.line, Column: l.column}
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenLessEqual, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenNotEqual, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else {
			tok = Token{Type: TokenLess, Literal: string(l.ch), Line: l.line, Column: l.column}
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = Token{Type: TokenGreaterEqual, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else {
			tok = Token{Type: TokenGreater, Literal: string(l.ch), Line: l.line, Column: l.column}
		}
	case ',':
		tok = Token{Type: TokenComma, Literal: string(l.ch), Line: l.line, Column: l.column}
	case ';':
		tok = Token{Type: TokenSemicolon, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '(':
		tok = Token{Type: TokenLParen, Literal: string(l.ch), Line: l.line, Column: l.column}
	case ')':
		tok = Token{Type: TokenRParen, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '*':
		tok = Token{Type: TokenStar, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '?':
		tok = Token{Type: TokenQuestion, Literal: string(l.ch), Line: l.line, Column: l.column}
	case '.':
		// Handle dot - could be part of qualified name (table.column) or standalone
		if isDigit(l.peekChar()) {
			// This is part of a float number - handle it in the number case
			tok = Token{Type: TokenIllegal, Literal: string(l.ch), Line: l.line, Column: l.column}
		} else {
			tok = Token{Type: TokenDot, Literal: string(l.ch), Line: l.line, Column: l.column}
		}
	case '\'', '"':
		tok.Type = TokenString
		tok.Literal = l.readString(l.ch)
		tok.Line = l.line
		tok.Column = l.column
	case 0:
		tok.Literal = ""
		tok.Type = TokenEOF
		tok.Line = l.line
		tok.Column = l.column
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = lookupIdent(tok.Literal)
			tok.Line = l.line
			tok.Column = l.column
			return tok // early return to avoid l.readChar()
		} else if isDigit(l.ch) {
			literal, tokenType := l.readNumber()
			tok.Type = tokenType
			tok.Literal = literal
			tok.Line = l.line
			tok.Column = l.column
			return tok // early return to avoid l.readChar()
		} else {
			tok = Token{Type: TokenIllegal, Literal: string(l.ch), Line: l.line, Column: l.column}
		}
	}

	l.readChar()
	return tok
}

// lookupIdent checks if identifier is a keyword
func lookupIdent(ident string) TokenType {
	if tok, ok := keywords[strings.ToUpper(ident)]; ok {
		return tok
	}
	return TokenIdent
}

// Helper functions

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

// DetectSQL detects if the input string is a SQL statement
func DetectSQL(input string) bool {
	if input == "" {
		return false
	}

	// Try to parse as JSON first
	trimmed := strings.TrimSpace(input)
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		return false // Likely JSON
	}

	// Check for SQL keywords at the beginning
	lexer := NewLexer(input)
	tok := lexer.NextToken()

	return tok.Type == TokenSelect || tok.Type == TokenInsert ||
		tok.Type == TokenUpdate || tok.Type == TokenDelete
}

// TokenTypeString returns string representation of token type
func (t TokenType) String() string {
	switch t {
	case TokenIllegal:
		return "ILLEGAL"
	case TokenEOF:
		return "EOF"
	case TokenIdent:
		return "IDENT"
	case TokenInt:
		return "INT"
	case TokenFloat:
		return "FLOAT"
	case TokenString:
		return "STRING"
	case TokenSelect:
		return "SELECT"
	case TokenFrom:
		return "FROM"
	case TokenWhere:
		return "WHERE"
	case TokenInsert:
		return "INSERT"
	case TokenInto:
		return "INTO"
	case TokenValues:
		return "VALUES"
	case TokenUpdate:
		return "UPDATE"
	case TokenSet:
		return "SET"
	case TokenDelete:
		return "DELETE"
	case TokenOrder:
		return "ORDER"
	case TokenBy:
		return "BY"
	case TokenGroup:
		return "GROUP"
	case TokenHaving:
		return "HAVING"
	case TokenLimit:
		return "LIMIT"
	case TokenOffset:
		return "OFFSET"
	case TokenJoin:
		return "JOIN"
	case TokenInner:
		return "INNER"
	case TokenLeft:
		return "LEFT"
	case TokenRight:
		return "RIGHT"
	case TokenFull:
		return "FULL"
	case TokenOn:
		return "ON"
	case TokenAs:
		return "AS"
	case TokenAnd:
		return "AND"
	case TokenOr:
		return "OR"
	case TokenNot:
		return "NOT"
	case TokenIn:
		return "IN"
	case TokenIs:
		return "IS"
	case TokenNull:
		return "NULL"
	case TokenLike:
		return "LIKE"
	case TokenBetween:
		return "BETWEEN"
	case TokenDistinct:
		return "DISTINCT"
	case TokenTrue:
		return "TRUE"
	case TokenFalse:
		return "FALSE"
	case TokenEqual:
		return "="
	case TokenNotEqual:
		return "!="
	case TokenLess:
		return "<"
	case TokenLessEqual:
		return "<="
	case TokenGreater:
		return ">"
	case TokenGreaterEqual:
		return ">="
	case TokenComma:
		return ","
	case TokenSemicolon:
		return ";"
	case TokenLParen:
		return "("
	case TokenRParen:
		return ")"
	case TokenStar:
		return "*"
	case TokenQuestion:
		return "?"
	case TokenDot:
		return "."
	default:
		return fmt.Sprintf("UNKNOWN(%d)", int(t))
	}
}
