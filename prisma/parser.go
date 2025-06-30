package prisma

import (
	"fmt"
)

// Parser represents the parser
type Parser struct {
	lexer *Lexer

	curToken  Token
	peekToken Token

	errors []string
}

// New creates a new parser instance
func NewParser(lexer *Lexer) *Parser {
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

// ParseSchema parses the schema and returns AST
func (p *Parser) ParseSchema() *PrismaSchema {
	schema := &PrismaSchema{}
	schema.Statements = []Statement{}

	for p.curToken.Type != EOF {
		// Skip comments
		if p.curToken.Type == COMMENT {
			p.nextToken()
			continue
		}

		stmt := p.parseStatement()
		if stmt != nil {
			schema.Statements = append(schema.Statements, stmt)
		}
		p.nextToken()
	}

	return schema
}

// parseStatement parses a statement
func (p *Parser) parseStatement() Statement {
	switch p.curToken.Type {
	case MODEL:
		return p.parseModelStatement()
	case ENUM:
		return p.parseEnumStatement()
	case DATASOURCE:
		return p.parseDatasourceStatement()
	case GENERATOR:
		return p.parseGeneratorStatement()
	default:
		p.addError(fmt.Sprintf("unexpected token %s at line %d", p.curToken.Type, p.curToken.Line))
		return nil
	}
}

// parseModelStatement parses a model statement
func (p *Parser) parseModelStatement() *ModelStatement {
	stmt := &ModelStatement{}

	if !p.expectPeek(IDENT) {
		return nil
	}

	stmt.Name = p.curToken.Literal

	if !p.expectPeek(LBRACE) {
		return nil
	}

	stmt.Fields = p.parseFields()
	stmt.BlockAttributes = p.parseBlockAttributes()

	// parseFields should have left us positioned right before RBRACE
	if p.curToken.Type != RBRACE {
		p.addError(fmt.Sprintf("expected }, got %s at line %d", p.curToken.Type, p.curToken.Line))
		return nil
	}

	return stmt
}

// parseEnumStatement parses an enum statement
func (p *Parser) parseEnumStatement() *EnumStatement {
	stmt := &EnumStatement{}

	if !p.expectPeek(IDENT) {
		return nil
	}

	stmt.Name = p.curToken.Literal

	if !p.expectPeek(LBRACE) {
		return nil
	}

	stmt.Values = p.parseEnumValues()

	// parseEnumValues should have left us positioned at RBRACE
	if p.curToken.Type != RBRACE {
		p.addError(fmt.Sprintf("expected }, got %s at line %d", p.curToken.Type, p.curToken.Line))
		return nil
	}

	return stmt
}

// parseDatasourceStatement parses a datasource statement
func (p *Parser) parseDatasourceStatement() *DatasourceStatement {
	stmt := &DatasourceStatement{}

	// datasource name can be an identifier or keyword like "db"
	p.nextToken() // advance to name token
	if p.curToken.Type == IDENT || p.curToken.Type == DB {
		stmt.Name = p.curToken.Literal
	} else {
		p.addError(fmt.Sprintf("expected datasource name, got %s at line %d", p.curToken.Type, p.curToken.Line))
		return nil
	}

	if !p.expectPeek(LBRACE) {
		return nil
	}

	stmt.Properties = p.parseProperties()

	// parseProperties should have left us positioned at RBRACE
	if p.curToken.Type != RBRACE {
		p.addError(fmt.Sprintf("expected }, got %s at line %d", p.curToken.Type, p.curToken.Line))
		return nil
	}

	return stmt
}

// parseGeneratorStatement parses a generator statement (but we don't use it for code generation)
func (p *Parser) parseGeneratorStatement() *GeneratorStatement {
	stmt := &GeneratorStatement{}

	if !p.expectPeek(IDENT) {
		return nil
	}

	stmt.Name = p.curToken.Literal

	if !p.expectPeek(LBRACE) {
		return nil
	}

	stmt.Properties = p.parseProperties()

	// parseProperties should have left us positioned at RBRACE
	if p.curToken.Type != RBRACE {
		p.addError(fmt.Sprintf("expected }, got %s at line %d", p.curToken.Type, p.curToken.Line))
		return nil
	}

	return stmt
}

// parseFields parses model fields
func (p *Parser) parseFields() []*Field {
	fields := []*Field{}

	p.nextToken()

	for p.curToken.Type != RBRACE && p.curToken.Type != EOF {
		if p.curToken.Type == BLOCK_AT {
			// This is a block attribute, break to handle it separately
			break
		}

		if p.curToken.Type == IDENT {
			field := p.parseField()
			if field != nil {
				fields = append(fields, field)
			}
		}
		p.nextToken()
	}

	return fields
}

// parseField parses a single field
func (p *Parser) parseField() *Field {
	field := &Field{}
	field.Name = p.curToken.Literal

	// Advance to the type token
	if p.isTypeToken(p.peekToken.Type) {
		p.nextToken() // consume the type token
	} else if !p.expectPeek(IDENT) {
		return nil
	}

	field.Type = &FieldType{Name: p.curToken.Literal}

	// Check for array type
	if p.peekToken.Type == LBRACKET {
		p.nextToken() // consume '['
		if !p.expectPeek(RBRACKET) {
			return nil
		}
		field.List = true
	}

	// Check for optional
	if p.peekToken.Type == QUESTION {
		p.nextToken()
		field.Optional = true
	}

	// Parse attributes
	field.Attributes = p.parseAttributes()

	return field
}

// parseAttributes parses field attributes
func (p *Parser) parseAttributes() []*Attribute {
	attributes := []*Attribute{}

	for p.peekToken.Type == AT {
		p.nextToken() // consume '@'
		p.nextToken() // advance to attribute name
		
		// Handle both IDENT and keyword tokens as attribute names
		if p.curToken.Type != IDENT && !p.isAttributeKeyword(p.curToken.Type) {
			p.addError(fmt.Sprintf("expected attribute name, got %s at line %d", p.curToken.Type, p.curToken.Line))
			break
		}

		attrName := p.curToken.Literal

		// Check if it's a dot notation attribute (like @db.VarChar)
		if p.peekToken.Type == DOT {
			p.nextToken() // consume '.'
			p.nextToken() // advance to property name
			if p.curToken.Type != IDENT && !p.isAttributeKeyword(p.curToken.Type) {
				p.addError(fmt.Sprintf("expected identifier after dot in attribute, got %s at line %d", p.curToken.Type, p.curToken.Line))
				break
			}
			attrName += "." + p.curToken.Literal
		}

		attr := &Attribute{Name: attrName}

		// Check for arguments
		if p.peekToken.Type == LPAREN {
			p.nextToken() // consume '('
			attr.Args = p.parseArgumentList()
			// parseArgumentList should leave us at the closing parenthesis
			if p.curToken.Type != RPAREN {
				p.addError(fmt.Sprintf("expected ), got %s at line %d", p.curToken.Type, p.curToken.Line))
				break
			}
		}

		attributes = append(attributes, attr)
	}

	return attributes
}

// parseBlockAttributes parses block-level attributes
func (p *Parser) parseBlockAttributes() []*BlockAttribute {
	attributes := []*BlockAttribute{}

	for p.curToken.Type == BLOCK_AT {
		p.nextToken() // advance to attribute name
		
		if p.curToken.Type != IDENT && !p.isAttributeKeyword(p.curToken.Type) {
			p.addError(fmt.Sprintf("expected block attribute name, got %s at line %d", p.curToken.Type, p.curToken.Line))
			break
		}

		attrName := p.curToken.Literal

		// Check if it's a dot notation attribute (like @@db.something)
		if p.peekToken.Type == DOT {
			p.nextToken() // consume '.'
			p.nextToken() // advance to property name
			if p.curToken.Type != IDENT && !p.isAttributeKeyword(p.curToken.Type) {
				p.addError(fmt.Sprintf("expected identifier after dot in block attribute, got %s at line %d", p.curToken.Type, p.curToken.Line))
				break
			}
			attrName += "." + p.curToken.Literal
		}

		attr := &BlockAttribute{Name: attrName}

		// Check for arguments
		if p.peekToken.Type == LPAREN {
			p.nextToken() // consume '('
			attr.Args = p.parseArgumentList()
			// parseArgumentList should leave us at the closing parenthesis
			if p.curToken.Type != RPAREN {
				p.addError(fmt.Sprintf("expected ), got %s at line %d", p.curToken.Type, p.curToken.Line))
				break
			}
		}

		attributes = append(attributes, attr)
		p.nextToken()
	}

	return attributes
}

// parseArgumentList parses a list of arguments
func (p *Parser) parseArgumentList() []Expression {
	args := []Expression{}

	if p.peekToken.Type == RPAREN {
		p.nextToken() // consume ')'
		return args
	}

	p.nextToken()
	args = append(args, p.parseExpression())

	for p.peekToken.Type == COMMA {
		p.nextToken() // consume ','
		p.nextToken()
		args = append(args, p.parseExpression())
	}

	// Advance to the closing parenthesis
	if p.peekToken.Type == RPAREN {
		p.nextToken() // consume ')'
	}

	return args
}

// parseExpression parses an expression
func (p *Parser) parseExpression() Expression {
	switch p.curToken.Type {
	case IDENT:
		// Check if it's a function call
		if p.peekToken.Type == LPAREN {
			name := p.curToken.Literal
			p.nextToken() // consume '('
			fc := &FunctionCall{Name: name}
			fc.Args = p.parseArgumentList()
			return fc
		}
		// Check if it's a named argument (key: value)
		if p.peekToken.Type == COLON {
			name := p.curToken.Literal
			p.nextToken() // consume ':'
			p.nextToken() // advance to value
			value := p.parseExpression()
			return &NamedArgument{Name: name, Value: value}
		}
		// Check if it's a dot expression (obj.property or obj.function())
		if p.peekToken.Type == DOT {
			left := &Identifier{Value: p.curToken.Literal}
			p.nextToken() // consume '.'
			p.nextToken() // advance to property name
			if p.curToken.Type != IDENT {
				p.addError(fmt.Sprintf("expected identifier after dot, got %s at line %d", p.curToken.Type, p.curToken.Line))
				return nil
			}
			
			// Check if the property is a function call
			if p.peekToken.Type == LPAREN {
				funcName := p.curToken.Literal
				p.nextToken() // consume '('
				fc := &FunctionCall{Name: funcName}
				fc.Args = p.parseArgumentList()
				return &DotExpression{Left: left, Right: fc.String()}
			}
			
			return &DotExpression{Left: left, Right: p.curToken.Literal}
		}
		return &Identifier{Value: p.curToken.Literal}
	case STRING:
		return &StringLiteral{Value: p.curToken.Literal}
	case NUMBER:
		return &NumberLiteral{Value: p.curToken.Literal}
	case LBRACKET:
		return p.parseArrayExpression()
	case TRUE, FALSE:
		return &Identifier{Value: p.curToken.Literal}
	case DEFAULT, AUTOINCREMENT, NOW, UUID, ENV, UPDATEDAT, CUID, DB, TEXT, VARCHAR, MONEY, JSONB, UUID_TYPE, TIMESTAMP, DATE_TYPE, TIME_TYPE, DECIMAL_TYPE, DOUBLEPRECISION, REAL, SMALLINT, BIGINT_TYPE, SERIAL, BIGSERIAL, CHAR, INET, BIT, VARBIT, XML, CASCADE, RESTRICT, NOACTION, SETNULL, SETDEFAULT, ONDELETE, ONUPDATE, DBGENERATED:
		// These are treated as function calls when they appear as identifiers
		if p.peekToken.Type == LPAREN {
			name := p.curToken.Literal
			p.nextToken() // consume '('
			fc := &FunctionCall{Name: name}
			fc.Args = p.parseArgumentList()
			return fc
		}
		// Check if it's a named argument (key: value)
		if p.peekToken.Type == COLON {
			name := p.curToken.Literal
			p.nextToken() // consume ':'
			p.nextToken() // advance to value
			value := p.parseExpression()
			return &NamedArgument{Name: name, Value: value}
		}
		// Check if DB is used in dot notation (db.VarChar)
		if p.curToken.Type == DB && p.peekToken.Type == DOT {
			left := &Identifier{Value: p.curToken.Literal}
			p.nextToken() // consume '.'
			p.nextToken() // advance to property name
			if p.curToken.Type != IDENT {
				p.addError(fmt.Sprintf("expected identifier after dot, got %s at line %d", p.curToken.Type, p.curToken.Line))
				return nil
			}
			
			// Check if the property is a function call
			if p.peekToken.Type == LPAREN {
				funcName := p.curToken.Literal
				p.nextToken() // consume '('
				fc := &FunctionCall{Name: funcName}
				fc.Args = p.parseArgumentList()
				return &DotExpression{Left: left, Right: fc.String()}
			}
			
			return &DotExpression{Left: left, Right: p.curToken.Literal}
		}
		return &Identifier{Value: p.curToken.Literal}
	default:
		p.addError(fmt.Sprintf("unexpected token in expression: %s", p.curToken.Type))
		return nil
	}
}

// parseArrayExpression parses an array expression
func (p *Parser) parseArrayExpression() Expression {
	arr := &ArrayExpression{}

	if p.peekToken.Type == RBRACKET {
		p.nextToken() // consume ']'
		return arr
	}

	p.nextToken()
	arr.Elements = append(arr.Elements, p.parseExpression())

	for p.peekToken.Type == COMMA {
		p.nextToken() // consume ','
		p.nextToken()
		arr.Elements = append(arr.Elements, p.parseExpression())
	}

	if !p.expectPeek(RBRACKET) {
		return nil
	}

	return arr
}

// parseEnumValues parses enum values with optional attributes
func (p *Parser) parseEnumValues() []*EnumValue {
	values := []*EnumValue{}

	p.nextToken()

	for p.curToken.Type != RBRACE && p.curToken.Type != EOF {
		if p.curToken.Type == IDENT {
			enumValue := &EnumValue{
				Name: p.curToken.Literal,
			}
			
			// Parse attributes for this enum value
			enumValue.Attributes = p.parseAttributes()
			
			values = append(values, enumValue)
		}
		p.nextToken() // Always advance token to avoid infinite loop
	}

	return values
}

// parseProperties parses datasource/generator properties
func (p *Parser) parseProperties() []*Property {
	properties := []*Property{}

	p.nextToken()

	for p.curToken.Type != RBRACE && p.curToken.Type != EOF {
		if p.curToken.Type == IDENT {
			prop := &Property{Name: p.curToken.Literal}

			if !p.expectPeek(EQUALS) {
				break
			}

			p.nextToken() // Move to the value token
			prop.Value = p.parseExpression()
			properties = append(properties, prop)
		}
		p.nextToken()
	}

	return properties
}

// expectPeek checks the peek token type and advances if it matches
func (p *Parser) expectPeek(t TokenType) bool {
	if p.peekToken.Type == t {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

// peekError adds an error for unexpected peek token
func (p *Parser) peekError(t TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead at line %d", 
		t, p.peekToken.Type, p.peekToken.Line)
	p.errors = append(p.errors, msg)
}

// addError adds an error message
func (p *Parser) addError(msg string) {
	p.errors = append(p.errors, msg)
}

// isTypeToken checks if token is a type token
func (p *Parser) isTypeToken(tokenType TokenType) bool {
	switch tokenType {
	case INT, STRING_TYPE, BOOLEAN, DATETIME, JSON, FLOAT, DECIMAL:
		return true
	default:
		return false
	}
}

// isAttributeKeyword checks if token can be used as an attribute name
func (p *Parser) isAttributeKeyword(tokenType TokenType) bool {
	switch tokenType {
	case DEFAULT, AUTOINCREMENT, NOW, UUID, ENV, UPDATEDAT, CUID, DB, TEXT, VARCHAR, MONEY, JSONB, UUID_TYPE, TIMESTAMP, DATE_TYPE, TIME_TYPE, DECIMAL_TYPE, DOUBLEPRECISION, REAL, SMALLINT, BIGINT_TYPE, SERIAL, BIGSERIAL, CHAR, INET, BIT, VARBIT, XML, DECIMAL:
		return true
	default:
		return false
	}
}