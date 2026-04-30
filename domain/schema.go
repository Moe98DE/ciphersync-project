package domain

import "context"

// Column represents a single field within a database table.
type Column struct {
	Name       string
	DataType   string
	IsNullable bool
}

// ForeignKey represents a directional edge in our DAG.
// It maps exactly to what we need to safely toggle constraints (Milestone 3).
type ForeignKey struct {
	ConstraintName string
	SourceColumn   string
	TargetTable    string
	TargetColumn   string
}

// Table represents a node in the DAG.
type Table struct {
	Schema      string
	Name        string
	Columns     []Column
	PrimaryKeys []string
	ForeignKeys []ForeignKey
}

// Introspector is the contract Team Beta must satisfy for every database dialect.
// Team Alpha will use this to build their in-memory DAG.
type Introspector interface {
	// Introspect connects to the underlying database and maps the complete schema.
	Introspect(ctx context.Context) ([]*Table, error)
}
