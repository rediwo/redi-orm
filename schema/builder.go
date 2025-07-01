package schema

type FieldBuilder struct {
	field Field
}

func NewField(name string) *FieldBuilder {
	return &FieldBuilder{
		field: Field{
			Name:     name,
			Type:     FieldTypeString,
			Nullable: false,
		},
	}
}

func (fb *FieldBuilder) String() *FieldBuilder {
	fb.field.Type = FieldTypeString
	return fb
}

func (fb *FieldBuilder) Int() *FieldBuilder {
	fb.field.Type = FieldTypeInt
	return fb
}

func (fb *FieldBuilder) Int64() *FieldBuilder {
	fb.field.Type = FieldTypeInt64
	return fb
}

func (fb *FieldBuilder) Float() *FieldBuilder {
	fb.field.Type = FieldTypeFloat
	return fb
}

func (fb *FieldBuilder) Bool() *FieldBuilder {
	fb.field.Type = FieldTypeBool
	return fb
}

func (fb *FieldBuilder) DateTime() *FieldBuilder {
	fb.field.Type = FieldTypeDateTime
	return fb
}

func (fb *FieldBuilder) JSON() *FieldBuilder {
	fb.field.Type = FieldTypeJSON
	return fb
}

func (fb *FieldBuilder) PrimaryKey() *FieldBuilder {
	fb.field.PrimaryKey = true
	fb.field.Nullable = false
	return fb
}

func (fb *FieldBuilder) AutoIncrement() *FieldBuilder {
	fb.field.AutoIncrement = true
	return fb
}

func (fb *FieldBuilder) Nullable() *FieldBuilder {
	fb.field.Nullable = true
	return fb
}

func (fb *FieldBuilder) Unique() *FieldBuilder {
	fb.field.Unique = true
	return fb
}

func (fb *FieldBuilder) Default(value interface{}) *FieldBuilder {
	fb.field.Default = value
	return fb
}

func (fb *FieldBuilder) Index() *FieldBuilder {
	fb.field.Index = true
	return fb
}

func (fb *FieldBuilder) Map(columnName string) *FieldBuilder {
	fb.field.Map = columnName
	return fb
}

func (fb *FieldBuilder) DbType(dbType string) *FieldBuilder {
	fb.field.DbType = dbType
	return fb
}

func (fb *FieldBuilder) Build() Field {
	return fb.field
}
