package types

// IncludeOption represents options for including relations
type IncludeOption struct {
	// The relation path (e.g., "posts" or "posts.comments")
	Path string

	// Fields to select from the related model (nil means all fields)
	Select []string

	// Where conditions to filter the related records
	Where Condition

	// Order by for the related records
	OrderBy []OrderByOption

	// Limit for the related records
	Limit *int

	// Offset for the related records
	Offset *int
}

// OrderByOption represents ordering option for includes
type OrderByOption struct {
	Field     string
	Direction Order
}

// IncludeOptions represents a collection of include options
type IncludeOptions map[string]*IncludeOption
