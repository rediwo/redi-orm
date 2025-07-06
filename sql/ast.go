package sql

// StatementType represents the type of SQL statement
type StatementType int

const (
	StatementTypeSelect StatementType = iota
	StatementTypeInsert
	StatementTypeUpdate
	StatementTypeDelete
)

// SQLStatement represents a parsed SQL statement
type SQLStatement interface {
	GetType() StatementType
	GetTableName() string
}

// SelectField represents a field in SELECT clause
type SelectField struct {
	Expression string // Field name or expression
	Alias      string // Optional alias
}

// TableRef represents a table reference
type TableRef struct {
	Table string // Table name
	Alias string // Optional alias
}

// JoinType represents the type of JOIN
type JoinType int

const (
	JoinTypeInner JoinType = iota
	JoinTypeLeft
	JoinTypeRight
	JoinTypeFull
)

// JoinClause represents a JOIN clause
type JoinClause struct {
	Type      JoinType
	Table     TableRef
	Condition *WhereClause
}

// OrderDirection represents sort direction
type OrderDirection int

const (
	OrderDirectionAsc OrderDirection = iota
	OrderDirectionDesc
)

// OrderByClause represents an ORDER BY clause
type OrderByClause struct {
	Field     string
	Direction OrderDirection
}

// WhereClause represents a WHERE condition
type WhereClause struct {
	Operator  string       // "AND", "OR", "NOT"
	Left      *WhereClause // Left operand for logical operators
	Right     *WhereClause // Right operand for logical operators
	Condition *Condition   // Leaf condition
}

// Condition represents a single condition
type Condition struct {
	Field    string           // Field name
	Operator string           // "=", ">", "<", ">=", "<=", "!=", "LIKE", "IN", "NOT IN", "IS NULL", "IS NOT NULL"
	Value    any              // Single value
	Values   []any            // Multiple values for IN clause
	Subquery *SelectStatement // Subquery for IN/EXISTS clauses
}

// SelectStatement represents a SELECT statement
type SelectStatement struct {
	Fields  []SelectField
	From    TableRef
	Where   *WhereClause
	OrderBy []*OrderByClause
	GroupBy []string
	Having  *WhereClause
	Limit   *int
	Offset  *int
	Joins   []*JoinClause
}

func (s *SelectStatement) GetType() StatementType {
	return StatementTypeSelect
}

func (s *SelectStatement) GetTableName() string {
	return s.From.Table
}

// InsertStatement represents an INSERT statement
type InsertStatement struct {
	Table  string
	Fields []string
	Values [][]any // Multiple rows of values
}

func (s *InsertStatement) GetType() StatementType {
	return StatementTypeInsert
}

func (s *InsertStatement) GetTableName() string {
	return s.Table
}

// UpdateStatement represents an UPDATE statement
type UpdateStatement struct {
	Table string
	Set   map[string]any
	Where *WhereClause
}

func (s *UpdateStatement) GetType() StatementType {
	return StatementTypeUpdate
}

func (s *UpdateStatement) GetTableName() string {
	return s.Table
}

// DeleteStatement represents a DELETE statement
type DeleteStatement struct {
	Table string
	Where *WhereClause
}

func (s *DeleteStatement) GetType() StatementType {
	return StatementTypeDelete
}

func (s *DeleteStatement) GetTableName() string {
	return s.Table
}

// Helper functions

// IsSelectAll checks if the SELECT fields represent SELECT *
func IsSelectAll(fields []SelectField) bool {
	return len(fields) == 1 && fields[0].Expression == "*"
}

// NewCondition creates a new simple condition
func NewCondition(field, operator string, value any) *Condition {
	return &Condition{
		Field:    field,
		Operator: operator,
		Value:    value,
	}
}

// NewInCondition creates a new IN condition
func NewInCondition(field string, values []any) *Condition {
	return &Condition{
		Field:    field,
		Operator: "IN",
		Values:   values,
	}
}

// NewWhereClause creates a new WHERE clause with a single condition
func NewWhereClause(condition *Condition) *WhereClause {
	return &WhereClause{
		Condition: condition,
	}
}

// And combines two WHERE clauses with AND
func (w *WhereClause) And(other *WhereClause) *WhereClause {
	return &WhereClause{
		Operator: "AND",
		Left:     w,
		Right:    other,
	}
}

// Or combines two WHERE clauses with OR
func (w *WhereClause) Or(other *WhereClause) *WhereClause {
	return &WhereClause{
		Operator: "OR",
		Left:     w,
		Right:    other,
	}
}

// Not negates a WHERE clause
func (w *WhereClause) Not() *WhereClause {
	return &WhereClause{
		Operator: "NOT",
		Left:     w,
	}
}
