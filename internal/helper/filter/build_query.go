package helper

import "fmt"

type QueryBuilder struct {
	Conditions []string
	Args       []any
	Position   int
}

func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		Conditions: make([]string, 0),
		Args:       make([]any, 0),
		Position:   1,
	}
}

func (q *QueryBuilder) Add(condition string, value any) {
	q.Conditions = append(q.Conditions, fmt.Sprintf(condition, q.Position)) // "price id>$1",  1
	q.Args = append(q.Args, value)
	q.Position++
}
