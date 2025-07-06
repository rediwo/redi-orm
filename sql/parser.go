package sql

import (
	"fmt"
	"strconv"
	"strings"
)

// Parser represents the SQL parser
type Parser struct {
	lexer *Lexer

	curToken  Token
	peekToken Token

	errors []string
}

// NewParser creates a new parser instance
func NewParser(input string) *Parser {
	lexer := NewLexer(input)
	p := &Parser{
		lexer:  lexer,
		errors: []string{},
	}

	// Read two tokens, so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	return p
}

// nextToken advances the tokens
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

// Errors returns parsing errors
func (p *Parser) Errors() []string {
	return p.errors
}

// addError adds an error message
func (p *Parser) addError(msg string) {
	p.errors = append(p.errors, fmt.Sprintf("line %d, column %d: %s", p.curToken.Line, p.curToken.Column, msg))
}

// expectToken checks if current token matches expected type and advances
func (p *Parser) expectToken(t TokenType) bool {
	if p.curToken.Type == t {
		p.nextToken()
		return true
	}
	p.addError(fmt.Sprintf("expected %s, got %s", t.String(), p.curToken.Type.String()))
	return false
}

// Parse parses the SQL statement and returns AST
func (p *Parser) Parse() (SQLStatement, error) {
	var stmt SQLStatement

	switch p.curToken.Type {
	case TokenSelect:
		stmt = p.parseSelectStatement()
	case TokenInsert:
		stmt = p.parseInsertStatement()
	case TokenUpdate:
		stmt = p.parseUpdateStatement()
	case TokenDelete:
		stmt = p.parseDeleteStatement()
	default:
		p.addError(fmt.Sprintf("unexpected token: %s", p.curToken.Type.String()))
	}

	if len(p.errors) > 0 {
		return nil, fmt.Errorf("parse errors: %s", strings.Join(p.errors, "; "))
	}

	return stmt, nil
}

// parseSelectStatement parses a SELECT statement
func (p *Parser) parseSelectStatement() *SelectStatement {
	stmt := &SelectStatement{}

	if !p.expectToken(TokenSelect) {
		return nil
	}

	// Parse DISTINCT (optional)
	if p.curToken.Type == TokenDistinct {
		p.nextToken()
	}

	// Parse SELECT fields
	stmt.Fields = p.parseSelectFields()

	// Parse FROM clause
	if !p.expectToken(TokenFrom) {
		return nil
	}
	stmt.From = p.parseTableRef()

	// Parse JOIN clauses (if any)
	stmt.Joins = p.parseJoinClauses()

	// Parse optional clauses
	for p.curToken.Type != TokenEOF && p.curToken.Type != TokenSemicolon && p.curToken.Type != TokenRParen {
		switch p.curToken.Type {
		case TokenWhere:
			p.nextToken()
			stmt.Where = p.parseWhereClause()
		case TokenOrder:
			p.nextToken()
			if !p.expectToken(TokenBy) {
				return nil
			}
			stmt.OrderBy = p.parseOrderByClause()
		case TokenGroup:
			p.nextToken()
			if !p.expectToken(TokenBy) {
				return nil
			}
			stmt.GroupBy = p.parseGroupByClause()
		case TokenHaving:
			p.nextToken()
			stmt.Having = p.parseWhereClause()
		case TokenLimit:
			p.nextToken()
			stmt.Limit = p.parseLimit()
		case TokenOffset:
			p.nextToken()
			stmt.Offset = p.parseOffset()
		default:
			p.addError(fmt.Sprintf("unexpected token in SELECT: %s", p.curToken.Type.String()))
			return nil
		}
	}

	return stmt
}

// parseSelectFields parses the field list in SELECT
func (p *Parser) parseSelectFields() []SelectField {
	var fields []SelectField

	if p.curToken.Type == TokenStar {
		fields = append(fields, SelectField{Expression: "*"})
		p.nextToken()
	} else {
		fields = append(fields, p.parseSelectField())

		for p.curToken.Type == TokenComma {
			p.nextToken()
			fields = append(fields, p.parseSelectField())
		}
	}

	return fields
}

// parseSelectField parses a single field in SELECT
func (p *Parser) parseSelectField() SelectField {
	field := SelectField{}

	if p.curToken.Type == TokenIdent {
		fieldName := p.curToken.Literal
		p.nextToken()

		// Check for qualified name (table.column)
		if p.curToken.Type == TokenDot {
			p.nextToken() // consume dot
			if p.curToken.Type == TokenIdent {
				fieldName += "." + p.curToken.Literal
				p.nextToken()
			}
		}

		// Check if this is a function call
		if p.curToken.Type == TokenLParen {
			// Function call like COUNT(*)
			functionCall := fieldName + "("
			p.nextToken() // consume '('

			// Parse function arguments
			for p.curToken.Type != TokenRParen && p.curToken.Type != TokenEOF {
				if p.curToken.Type == TokenStar {
					functionCall += "*"
				} else {
					functionCall += p.curToken.Literal
				}
				p.nextToken()

				if p.curToken.Type == TokenComma {
					functionCall += ","
					p.nextToken()
				}
			}

			if p.curToken.Type == TokenRParen {
				functionCall += ")"
				p.nextToken() // consume ')'
			}

			field.Expression = functionCall
		} else {
			field.Expression = fieldName
		}

		// Check for alias
		if p.curToken.Type == TokenAs {
			p.nextToken()
			if p.curToken.Type == TokenIdent {
				field.Alias = p.curToken.Literal
				p.nextToken()
			}
		} else if p.curToken.Type == TokenIdent {
			// Implicit alias (without AS keyword)
			field.Alias = p.curToken.Literal
			p.nextToken()
		}
	}

	return field
}

// parseTableRef parses a table reference
func (p *Parser) parseTableRef() TableRef {
	ref := TableRef{}

	if p.curToken.Type == TokenIdent {
		ref.Table = p.curToken.Literal
		p.nextToken()

		// Check for alias
		if p.curToken.Type == TokenAs {
			p.nextToken()
			if p.curToken.Type == TokenIdent {
				ref.Alias = p.curToken.Literal
				p.nextToken()
			}
		} else if p.curToken.Type == TokenIdent {
			// Implicit alias
			ref.Alias = p.curToken.Literal
			p.nextToken()
		}
	}

	return ref
}

// parseWhereClause parses WHERE conditions
func (p *Parser) parseWhereClause() *WhereClause {
	return p.parseOrExpression()
}

// parseOrExpression parses OR expressions
func (p *Parser) parseOrExpression() *WhereClause {
	left := p.parseAndExpression()

	for p.curToken.Type == TokenOr {
		p.nextToken()
		right := p.parseAndExpression()
		left = &WhereClause{
			Operator: "OR",
			Left:     left,
			Right:    right,
		}
	}

	return left
}

// parseAndExpression parses AND expressions
func (p *Parser) parseAndExpression() *WhereClause {
	left := p.parseNotExpression()

	for p.curToken.Type == TokenAnd {
		p.nextToken()
		right := p.parseNotExpression()
		left = &WhereClause{
			Operator: "AND",
			Left:     left,
			Right:    right,
		}
	}

	return left
}

// parseNotExpression parses NOT expressions
func (p *Parser) parseNotExpression() *WhereClause {
	if p.curToken.Type == TokenNot {
		p.nextToken()
		expr := p.parseComparisonExpression()
		return &WhereClause{
			Operator: "NOT",
			Left:     expr,
		}
	}

	return p.parseComparisonExpression()
}

// parseComparisonExpression parses comparison expressions
func (p *Parser) parseComparisonExpression() *WhereClause {
	if p.curToken.Type == TokenLParen {
		p.nextToken()
		expr := p.parseOrExpression()
		if !p.expectToken(TokenRParen) {
			return nil
		}
		return expr
	}

	// Parse field name or function call
	if p.curToken.Type != TokenIdent {
		p.addError("expected field name or function")
		return nil
	}

	field := p.curToken.Literal
	p.nextToken()

	// Check for function call
	if p.curToken.Type == TokenLParen {
		// Function call like COUNT(*) or SUM(field)
		functionCall := field + "("
		p.nextToken() // consume '('

		// Parse function arguments
		for p.curToken.Type != TokenRParen && p.curToken.Type != TokenEOF {
			if p.curToken.Type == TokenStar {
				functionCall += "*"
			} else if p.curToken.Type == TokenIdent {
				// Handle qualified field names in function arguments
				fieldName := p.curToken.Literal
				p.nextToken()
				if p.curToken.Type == TokenDot {
					p.nextToken() // consume dot
					if p.curToken.Type == TokenIdent {
						fieldName += "." + p.curToken.Literal
						p.nextToken()
					}
				}
				functionCall += fieldName
				continue // Skip the regular nextToken at the end
			} else {
				functionCall += p.curToken.Literal
			}
			p.nextToken()

			if p.curToken.Type == TokenComma {
				functionCall += ","
				p.nextToken()
			}
		}

		if p.curToken.Type == TokenRParen {
			functionCall += ")"
			p.nextToken() // consume ')'
		}

		field = functionCall
	} else {
		// Check for qualified name (table.column)
		if p.curToken.Type == TokenDot {
			p.nextToken() // consume dot
			if p.curToken.Type == TokenIdent {
				field += "." + p.curToken.Literal
				p.nextToken()
			}
		}
	}

	// Parse operator and value
	condition := p.parseCondition(field)
	if condition == nil {
		return nil
	}

	return &WhereClause{Condition: condition}
}

// parseCondition parses a single condition
func (p *Parser) parseCondition(field string) *Condition {
	condition := &Condition{Field: field}

	switch p.curToken.Type {
	case TokenEqual:
		condition.Operator = "="
		p.nextToken()
		condition.Value = p.parseValue()
	case TokenNotEqual:
		condition.Operator = "!="
		p.nextToken()
		condition.Value = p.parseValue()
	case TokenLess:
		condition.Operator = "<"
		p.nextToken()
		condition.Value = p.parseValue()
	case TokenLessEqual:
		condition.Operator = "<="
		p.nextToken()
		condition.Value = p.parseValue()
	case TokenGreater:
		condition.Operator = ">"
		p.nextToken()
		condition.Value = p.parseValue()
	case TokenGreaterEqual:
		condition.Operator = ">="
		p.nextToken()
		condition.Value = p.parseValue()
	case TokenLike:
		condition.Operator = "LIKE"
		p.nextToken()
		condition.Value = p.parseValue()
	case TokenIn:
		condition.Operator = "IN"
		p.nextToken()
		// Check if this is a subquery or value list
		if p.curToken.Type == TokenLParen && p.peekToken.Type == TokenSelect {
			// This is a subquery: IN (SELECT ...)
			p.nextToken() // consume '('
			condition.Subquery = p.parseSelectStatement()
			if !p.expectToken(TokenRParen) {
				return nil
			}
		} else {
			// This is a value list: IN (1, 2, 3)
			condition.Values = p.parseInValues()
		}
	case TokenIs:
		p.nextToken()
		if p.curToken.Type == TokenNot {
			p.nextToken()
			if p.curToken.Type == TokenNull {
				condition.Operator = "IS NOT NULL"
				p.nextToken()
			}
		} else if p.curToken.Type == TokenNull {
			condition.Operator = "IS NULL"
			p.nextToken()
		}
	default:
		p.addError(fmt.Sprintf("unexpected operator: %s", p.curToken.Type.String()))
		return nil
	}

	return condition
}

// parseValue parses a single value
func (p *Parser) parseValue() any {
	switch p.curToken.Type {
	case TokenIdent:
		// Handle field references (including qualified names like table.column)
		fieldName := p.curToken.Literal
		p.nextToken()

		// Check for qualified name (table.column)
		if p.curToken.Type == TokenDot {
			p.nextToken() // consume dot
			if p.curToken.Type == TokenIdent {
				fieldName += "." + p.curToken.Literal
				p.nextToken()
			}
		}
		return fieldName
	case TokenString:
		value := p.curToken.Literal
		p.nextToken()
		return value
	case TokenInt:
		value, err := strconv.ParseInt(p.curToken.Literal, 10, 64)
		if err != nil {
			p.addError(fmt.Sprintf("invalid integer: %s", p.curToken.Literal))
			return nil
		}
		p.nextToken()
		return value
	case TokenFloat:
		value, err := strconv.ParseFloat(p.curToken.Literal, 64)
		if err != nil {
			p.addError(fmt.Sprintf("invalid float: %s", p.curToken.Literal))
			return nil
		}
		p.nextToken()
		return value
	case TokenQuestion:
		// Parameter placeholder
		p.nextToken()
		return "?"
	case TokenNull:
		p.nextToken()
		return nil
	case TokenTrue:
		p.nextToken()
		return true
	case TokenFalse:
		p.nextToken()
		return false
	default:
		p.addError(fmt.Sprintf("unexpected value: %s", p.curToken.Type.String()))
		return nil
	}
}

// parseInValues parses values for IN clause
func (p *Parser) parseInValues() []any {
	var values []any

	if !p.expectToken(TokenLParen) {
		return nil
	}

	if p.curToken.Type != TokenRParen {
		values = append(values, p.parseValue())

		for p.curToken.Type == TokenComma {
			p.nextToken()
			values = append(values, p.parseValue())
		}
	}

	if !p.expectToken(TokenRParen) {
		return nil
	}

	return values
}

// parseOrderByClause parses ORDER BY clause
func (p *Parser) parseOrderByClause() []*OrderByClause {
	var clauses []*OrderByClause

	clause := &OrderByClause{}
	if p.curToken.Type == TokenIdent {
		clause.Field = p.curToken.Literal
		p.nextToken()

		// Check for qualified name (table.column)
		if p.curToken.Type == TokenDot {
			p.nextToken() // consume dot
			if p.curToken.Type == TokenIdent {
				clause.Field += "." + p.curToken.Literal
				p.nextToken()
			}
		}

		// Check for ASC/DESC
		if p.curToken.Type == TokenIdent {
			if strings.ToUpper(p.curToken.Literal) == "ASC" {
				clause.Direction = OrderDirectionAsc
				p.nextToken()
			} else if strings.ToUpper(p.curToken.Literal) == "DESC" {
				clause.Direction = OrderDirectionDesc
				p.nextToken()
			}
		}
	}
	clauses = append(clauses, clause)

	// Parse additional ORDER BY fields
	for p.curToken.Type == TokenComma {
		p.nextToken()
		clause := &OrderByClause{}
		if p.curToken.Type == TokenIdent {
			clause.Field = p.curToken.Literal
			p.nextToken()

			// Check for qualified name (table.column)
			if p.curToken.Type == TokenDot {
				p.nextToken() // consume dot
				if p.curToken.Type == TokenIdent {
					clause.Field += "." + p.curToken.Literal
					p.nextToken()
				}
			}

			if p.curToken.Type == TokenIdent {
				if strings.ToUpper(p.curToken.Literal) == "ASC" {
					clause.Direction = OrderDirectionAsc
					p.nextToken()
				} else if strings.ToUpper(p.curToken.Literal) == "DESC" {
					clause.Direction = OrderDirectionDesc
					p.nextToken()
				}
			}
		}
		clauses = append(clauses, clause)
	}

	return clauses
}

// parseGroupByClause parses GROUP BY clause
func (p *Parser) parseGroupByClause() []string {
	var fields []string

	if p.curToken.Type == TokenIdent {
		fieldName := p.curToken.Literal
		p.nextToken()

		// Check for qualified name (table.column)
		if p.curToken.Type == TokenDot {
			p.nextToken() // consume dot
			if p.curToken.Type == TokenIdent {
				fieldName += "." + p.curToken.Literal
				p.nextToken()
			}
		}
		fields = append(fields, fieldName)

		for p.curToken.Type == TokenComma {
			p.nextToken()
			if p.curToken.Type == TokenIdent {
				fieldName := p.curToken.Literal
				p.nextToken()

				// Check for qualified name (table.column)
				if p.curToken.Type == TokenDot {
					p.nextToken() // consume dot
					if p.curToken.Type == TokenIdent {
						fieldName += "." + p.curToken.Literal
						p.nextToken()
					}
				}
				fields = append(fields, fieldName)
			}
		}
	}

	return fields
}

// parseLimit parses LIMIT clause
func (p *Parser) parseLimit() *int {
	if p.curToken.Type == TokenInt {
		value, err := strconv.Atoi(p.curToken.Literal)
		if err != nil {
			p.addError(fmt.Sprintf("invalid limit value: %s", p.curToken.Literal))
			return nil
		}
		p.nextToken()
		return &value
	}
	return nil
}

// parseOffset parses OFFSET clause
func (p *Parser) parseOffset() *int {
	if p.curToken.Type == TokenInt {
		value, err := strconv.Atoi(p.curToken.Literal)
		if err != nil {
			p.addError(fmt.Sprintf("invalid offset value: %s", p.curToken.Literal))
			return nil
		}
		p.nextToken()
		return &value
	}
	return nil
}

// parseInsertStatement parses INSERT statement
func (p *Parser) parseInsertStatement() *InsertStatement {
	stmt := &InsertStatement{}

	if !p.expectToken(TokenInsert) {
		return nil
	}

	if !p.expectToken(TokenInto) {
		return nil
	}

	// Parse table name
	if p.curToken.Type == TokenIdent {
		stmt.Table = p.curToken.Literal
		p.nextToken()
	} else {
		p.addError("expected table name")
		return nil
	}

	// Parse field list (optional)
	if p.curToken.Type == TokenLParen {
		p.nextToken()
		stmt.Fields = p.parseFieldList()
		if !p.expectToken(TokenRParen) {
			return nil
		}
	}

	// Parse VALUES
	if !p.expectToken(TokenValues) {
		return nil
	}

	// Parse value lists
	stmt.Values = p.parseValuesList()

	return stmt
}

// parseFieldList parses field names list
func (p *Parser) parseFieldList() []string {
	var fields []string

	if p.curToken.Type == TokenIdent {
		fields = append(fields, p.curToken.Literal)
		p.nextToken()

		for p.curToken.Type == TokenComma {
			p.nextToken()
			if p.curToken.Type == TokenIdent {
				fields = append(fields, p.curToken.Literal)
				p.nextToken()
			}
		}
	}

	return fields
}

// parseValuesList parses multiple value lists
func (p *Parser) parseValuesList() [][]any {
	var valuesList [][]any

	if p.curToken.Type == TokenLParen {
		valuesList = append(valuesList, p.parseSingleValuesList())

		for p.curToken.Type == TokenComma {
			p.nextToken()
			if p.curToken.Type == TokenLParen {
				valuesList = append(valuesList, p.parseSingleValuesList())
			}
		}
	}

	return valuesList
}

// parseSingleValuesList parses a single values list
func (p *Parser) parseSingleValuesList() []any {
	var values []any

	if !p.expectToken(TokenLParen) {
		return nil
	}

	if p.curToken.Type != TokenRParen {
		values = append(values, p.parseValue())

		for p.curToken.Type == TokenComma {
			p.nextToken()
			values = append(values, p.parseValue())
		}
	}

	if !p.expectToken(TokenRParen) {
		return nil
	}

	return values
}

// parseUpdateStatement parses UPDATE statement
func (p *Parser) parseUpdateStatement() *UpdateStatement {
	stmt := &UpdateStatement{
		Set: make(map[string]any),
	}

	if !p.expectToken(TokenUpdate) {
		return nil
	}

	// Parse table name
	if p.curToken.Type == TokenIdent {
		stmt.Table = p.curToken.Literal
		p.nextToken()
	} else {
		p.addError("expected table name")
		return nil
	}

	// Parse SET clause
	if !p.expectToken(TokenSet) {
		return nil
	}

	stmt.Set = p.parseSetClause()

	// Parse WHERE clause (optional)
	if p.curToken.Type == TokenWhere {
		p.nextToken()
		stmt.Where = p.parseWhereClause()
	}

	return stmt
}

// parseSetClause parses SET clause for UPDATE
func (p *Parser) parseSetClause() map[string]any {
	setMap := make(map[string]any)

	// Parse first assignment
	if p.curToken.Type == TokenIdent {
		field := p.curToken.Literal
		p.nextToken()
		if p.expectToken(TokenEqual) {
			setMap[field] = p.parseValue()
		}
	}

	// Parse additional assignments
	for p.curToken.Type == TokenComma {
		p.nextToken()
		if p.curToken.Type == TokenIdent {
			field := p.curToken.Literal
			p.nextToken()
			if p.expectToken(TokenEqual) {
				setMap[field] = p.parseValue()
			}
		}
	}

	return setMap
}

// parseDeleteStatement parses DELETE statement
func (p *Parser) parseDeleteStatement() *DeleteStatement {
	stmt := &DeleteStatement{}

	if !p.expectToken(TokenDelete) {
		return nil
	}

	if !p.expectToken(TokenFrom) {
		return nil
	}

	// Parse table name
	if p.curToken.Type == TokenIdent {
		stmt.Table = p.curToken.Literal
		p.nextToken()
	} else {
		p.addError("expected table name")
		return nil
	}

	// Parse WHERE clause (optional)
	if p.curToken.Type == TokenWhere {
		p.nextToken()
		stmt.Where = p.parseWhereClause()
	}

	return stmt
}

// parseJoinClauses parses JOIN clauses
func (p *Parser) parseJoinClauses() []*JoinClause {
	var joins []*JoinClause

	for p.isJoinToken() {
		join := p.parseJoinClause()
		if join != nil {
			joins = append(joins, join)
		}
	}

	return joins
}

// isJoinToken checks if current token starts a JOIN clause
func (p *Parser) isJoinToken() bool {
	return p.curToken.Type == TokenJoin ||
		p.curToken.Type == TokenInner ||
		p.curToken.Type == TokenLeft ||
		p.curToken.Type == TokenRight ||
		p.curToken.Type == TokenFull
}

// parseJoinClause parses a single JOIN clause
func (p *Parser) parseJoinClause() *JoinClause {
	join := &JoinClause{}

	// Parse JOIN type
	switch p.curToken.Type {
	case TokenJoin:
		join.Type = JoinTypeInner // Default JOIN is INNER JOIN
		p.nextToken()
	case TokenInner:
		join.Type = JoinTypeInner
		p.nextToken()
		if !p.expectToken(TokenJoin) {
			return nil
		}
	case TokenLeft:
		join.Type = JoinTypeLeft
		p.nextToken()
		if !p.expectToken(TokenJoin) {
			return nil
		}
	case TokenRight:
		join.Type = JoinTypeRight
		p.nextToken()
		if !p.expectToken(TokenJoin) {
			return nil
		}
	case TokenFull:
		join.Type = JoinTypeFull
		p.nextToken()
		if !p.expectToken(TokenJoin) {
			return nil
		}
	default:
		p.addError(fmt.Sprintf("unexpected token in JOIN: %s", p.curToken.Type.String()))
		return nil
	}

	// Parse table reference
	join.Table = p.parseTableRef()

	// Parse ON condition
	if !p.expectToken(TokenOn) {
		return nil
	}
	join.Condition = p.parseWhereClause()

	return join
}
