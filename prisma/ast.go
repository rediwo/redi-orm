package prisma

import "fmt"

// Node represents an AST node
type Node interface {
	String() string
}

// Statement represents a statement node
type Statement interface {
	Node
	statementNode()
}

// Expression represents an expression node
type Expression interface {
	Node
	expressionNode()
}

// PrismaSchema represents the root of the AST
type PrismaSchema struct {
	Statements []Statement
}

func (ps *PrismaSchema) String() string {
	var out string
	for _, s := range ps.Statements {
		out += s.String() + "\n"
	}
	return out
}

// ModelStatement represents a model definition
type ModelStatement struct {
	Name            string
	Fields          []*Field
	BlockAttributes []*BlockAttribute
}

func (ms *ModelStatement) statementNode() {}
func (ms *ModelStatement) String() string {
	out := fmt.Sprintf("model %s {\n", ms.Name)
	for _, field := range ms.Fields {
		out += "  " + field.String() + "\n"
	}
	for _, attr := range ms.BlockAttributes {
		out += "  " + attr.String() + "\n"
	}
	out += "}"
	return out
}

// EnumValue represents an enum value with optional mapping
type EnumValue struct {
	Name       string
	Attributes []*Attribute
}

// EnumStatement represents an enum definition
type EnumStatement struct {
	Name   string
	Values []*EnumValue
}

func (es *EnumStatement) statementNode() {}
func (es *EnumStatement) String() string {
	out := fmt.Sprintf("enum %s {\n", es.Name)
	for _, value := range es.Values {
		out += "  " + value.Name
		for _, attr := range value.Attributes {
			out += " " + attr.String()
		}
		out += "\n"
	}
	out += "}"
	return out
}

// DatasourceStatement represents a datasource definition
type DatasourceStatement struct {
	Name       string
	Properties []*Property
}

func (ds *DatasourceStatement) statementNode() {}
func (ds *DatasourceStatement) String() string {
	out := fmt.Sprintf("datasource %s {\n", ds.Name)
	for _, prop := range ds.Properties {
		out += "  " + prop.String() + "\n"
	}
	out += "}"
	return out
}

// GeneratorStatement represents a generator definition
type GeneratorStatement struct {
	Name       string
	Properties []*Property
}

func (gs *GeneratorStatement) statementNode() {}
func (gs *GeneratorStatement) String() string {
	out := fmt.Sprintf("generator %s {\n", gs.Name)
	for _, prop := range gs.Properties {
		out += "  " + prop.String() + "\n"
	}
	out += "}"
	return out
}

// Field represents a field in a model
type Field struct {
	Name       string
	Type       *FieldType
	Optional   bool
	List       bool
	Attributes []*Attribute
}

func (f *Field) String() string {
	out := f.Name + " " + f.Type.String()
	if f.List {
		out += "[]"
	}
	if f.Optional {
		out += "?"
	}
	for _, attr := range f.Attributes {
		out += " " + attr.String()
	}
	return out
}

// FieldType represents a field type
type FieldType struct {
	Name string
}

func (ft *FieldType) String() string {
	return ft.Name
}

// Attribute represents a field attribute
type Attribute struct {
	Name string
	Args []Expression
}

func (a *Attribute) String() string {
	out := "@" + a.Name
	if len(a.Args) > 0 {
		out += "("
		for i, arg := range a.Args {
			if i > 0 {
				out += ", "
			}
			out += arg.String()
		}
		out += ")"
	}
	return out
}

// BlockAttribute represents a block-level attribute (@@)
type BlockAttribute struct {
	Name string
	Args []Expression
}

func (ba *BlockAttribute) String() string {
	out := "@@" + ba.Name
	if len(ba.Args) > 0 {
		out += "("
		for i, arg := range ba.Args {
			if i > 0 {
				out += ", "
			}
			out += arg.String()
		}
		out += ")"
	}
	return out
}

// Property represents a property in datasource/generator
type Property struct {
	Name  string
	Value Expression
}

func (p *Property) String() string {
	return p.Name + " = " + p.Value.String()
}

// Identifier represents an identifier expression
type Identifier struct {
	Value string
}

func (i *Identifier) expressionNode() {}
func (i *Identifier) String() string  { return i.Value }

// StringLiteral represents a string literal expression
type StringLiteral struct {
	Value string
}

func (sl *StringLiteral) expressionNode() {}
func (sl *StringLiteral) String() string  { return fmt.Sprintf(`"%s"`, sl.Value) }

// NumberLiteral represents a number literal expression
type NumberLiteral struct {
	Value string
}

func (nl *NumberLiteral) expressionNode() {}
func (nl *NumberLiteral) String() string  { return nl.Value }

// FunctionCall represents a function call expression
type FunctionCall struct {
	Name string
	Args []Expression
}

func (fc *FunctionCall) expressionNode() {}
func (fc *FunctionCall) String() string {
	out := fc.Name + "("
	for i, arg := range fc.Args {
		if i > 0 {
			out += ", "
		}
		out += arg.String()
	}
	out += ")"
	return out
}

// ArrayExpression represents an array expression
type ArrayExpression struct {
	Elements []Expression
}

func (ae *ArrayExpression) expressionNode() {}
func (ae *ArrayExpression) String() string {
	out := "["
	for i, elem := range ae.Elements {
		if i > 0 {
			out += ", "
		}
		out += elem.String()
	}
	out += "]"
	return out
}

// NamedArgument represents a named argument expression (key: value)
type NamedArgument struct {
	Name  string
	Value Expression
}

func (na *NamedArgument) expressionNode() {}
func (na *NamedArgument) String() string {
	return na.Name + ": " + na.Value.String()
}

// DotExpression represents a dot notation expression (obj.property)
type DotExpression struct {
	Left  Expression
	Right string
}

func (de *DotExpression) expressionNode() {}
func (de *DotExpression) String() string {
	return de.Left.String() + "." + de.Right
}
