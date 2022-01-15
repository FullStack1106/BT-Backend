package rel

import (
	"strconv"
	"strings"
)

// Querier interface defines contract to be used for query builder.
type Querier interface {
	Build(*Query)
}

// Build for given table using given queriers.
func Build(table string, queriers ...Querier) Query {
	var (
		query = newQuery()
	)

	if len(queriers) > 0 {
		_, query.empty = queriers[0].(Query)
	}

	for _, querier := range queriers {
		// avoid using indirect call to avoid heap allocation
		switch q := querier.(type) {
		case Query:
			q.Build(&query)
		case JoinQuery:
			q.Build(&query)
		case FilterQuery:
			q.Build(&query)
		case GroupQuery:
			q.Build(&query)
		case SortQuery:
			q.Build(&query)
		case Offset:
			q.Build(&query)
		case Limit:
			q.Build(&query)
		case Lock:
			q.Build(&query)
		case Unscoped:
			q.Build(&query)
		case Reload:
			q.Build(&query)
		case SQLQuery:
			q.Build(&query)
		case Preload:
			q.Build(&query)
		case Cascade:
			q.Build(&query)
		}
	}

	if query.Table == "" {
		query.Table = table
	}

	return query
}

// Query defines information about query generated by query builder.
type Query struct {
	empty         bool // TODO: use bitmask to mark what is updated and use it when merging two queries
	Table         string
	SelectQuery   SelectQuery
	JoinQuery     []JoinQuery
	WhereQuery    FilterQuery
	GroupQuery    GroupQuery
	SortQuery     []SortQuery
	OffsetQuery   Offset
	LimitQuery    Limit
	LockQuery     Lock
	SQLQuery      SQLQuery
	UnscopedQuery Unscoped
	ReloadQuery   Reload
	CascadeQuery  Cascade
	PreloadQuery  []string
	UsePrimaryDb  bool
}

// Build query.
func (q Query) Build(query *Query) {
	if query.empty {
		*query = q
	} else {
		// manual merge
		if q.Table != "" {
			query.Table = q.Table
		}

		if q.SelectQuery.Fields != nil {
			query.SelectQuery = q.SelectQuery
		}

		query.JoinQuery = append(query.JoinQuery, q.JoinQuery...)

		if !q.WhereQuery.None() {
			query.WhereQuery = query.WhereQuery.And(q.WhereQuery)
		}

		if q.GroupQuery.Fields != nil {
			query.GroupQuery = q.GroupQuery
		}

		query.SortQuery = append(query.SortQuery, q.SortQuery...)

		if q.OffsetQuery != 0 {
			query.OffsetQuery = q.OffsetQuery
		}

		if q.LimitQuery != 0 {
			query.LimitQuery = q.LimitQuery
		}

		if q.LockQuery != "" {
			query.LockQuery = q.LockQuery
		}

		query.ReloadQuery = query.ReloadQuery || q.ReloadQuery
		query.CascadeQuery = query.CascadeQuery || q.CascadeQuery
		query.UsePrimaryDb = query.UsePrimaryDb || q.UsePrimaryDb
	}
}

// Select filter fields to be selected from database.
func (q Query) Select(fields ...string) Query {
	q.SelectQuery = NewSelect(fields...)
	return q
}

// From set the table to be used for query.
func (q Query) From(table string) Query {
	q.Table = table
	return q
}

// Distinct sets select query to be distinct.
func (q Query) Distinct() Query {
	q.SelectQuery.OnlyDistinct = true
	return q
}

// Join current table with other table.
func (q Query) Join(table string, filter ...FilterQuery) Query {
	return q.JoinOn(table, "", "", filter...)
}

// JoinOn current table with other table.
func (q Query) JoinOn(table string, from string, to string, filter ...FilterQuery) Query {
	return q.JoinWith("JOIN", table, from, to, filter...)
}

// JoinWith current table with other table with custom join mode.
func (q Query) JoinWith(mode string, table string, from string, to string, filter ...FilterQuery) Query {
	NewJoinWith(mode, table, from, to, filter...).Build(&q) // TODO: ensure this always called last

	return q
}

// Joinf create join query using a raw query.
func (q Query) Joinf(expr string, args ...interface{}) Query {
	NewJoinFragment(expr, args...).Build(&q) // TODO: ensure this always called last

	return q
}

// Where query.
func (q Query) Where(filters ...FilterQuery) Query {
	q.WhereQuery = q.WhereQuery.And(filters...)
	return q
}

// Wheref create where query using a raw query.
func (q Query) Wheref(expr string, args ...interface{}) Query {
	q.WhereQuery = q.WhereQuery.And(FilterFragment(expr, args...))
	return q
}

// OrWhere query.
func (q Query) OrWhere(filters ...FilterQuery) Query {
	q.WhereQuery = q.WhereQuery.Or(And(filters...))
	return q
}

// OrWheref create where query using a raw query.
func (q Query) OrWheref(expr string, args ...interface{}) Query {
	q.WhereQuery = q.WhereQuery.Or(FilterFragment(expr, args...))
	return q
}

// Group query.
func (q Query) Group(fields ...string) Query {
	q.GroupQuery.Fields = fields
	return q
}

// Having query.
func (q Query) Having(filters ...FilterQuery) Query {
	q.GroupQuery.Filter = q.GroupQuery.Filter.And(filters...)
	return q
}

// Havingf create having query using a raw query.
func (q Query) Havingf(expr string, args ...interface{}) Query {
	q.GroupQuery.Filter = q.GroupQuery.Filter.And(FilterFragment(expr, args...))
	return q
}

// OrHaving query.
func (q Query) OrHaving(filters ...FilterQuery) Query {
	q.GroupQuery.Filter = q.GroupQuery.Filter.Or(And(filters...))
	return q
}

// OrHavingf create having query using a raw query.
func (q Query) OrHavingf(expr string, args ...interface{}) Query {
	q.GroupQuery.Filter = q.GroupQuery.Filter.Or(FilterFragment(expr, args...))
	return q
}

// Sort query.
func (q Query) Sort(fields ...string) Query {
	return q.SortAsc(fields...)
}

// SortAsc query.
func (q Query) SortAsc(fields ...string) Query {
	var (
		offset = len(q.SortQuery)
	)

	q.SortQuery = append(q.SortQuery, make([]SortQuery, len(fields))...)
	for i := range fields {
		q.SortQuery[offset+i] = NewSortAsc(fields[i])
	}

	return q
}

// SortDesc query.
func (q Query) SortDesc(fields ...string) Query {
	var (
		offset = len(q.SortQuery)
	)

	q.SortQuery = append(q.SortQuery, make([]SortQuery, len(fields))...)
	for i := range fields {
		q.SortQuery[offset+i] = NewSortDesc(fields[i])
	}

	return q
}

// Offset the result returned by database.
func (q Query) Offset(offset int) Query {
	q.OffsetQuery = Offset(offset)
	return q
}

// Limit result returned by database.
func (q Query) Limit(limit int) Query {
	q.LimitQuery = Limit(limit)
	return q
}

// Lock query expression.
func (q Query) Lock(lock string) Query {
	q.LockQuery = Lock(lock)
	return q
}

// Unscoped allows soft-delete to be ignored.
func (q Query) Unscoped() Query {
	q.UnscopedQuery = true
	return q
}

// Reload force reloading association on preload.
func (q Query) Reload() Query {
	q.ReloadQuery = true
	return q
}

// Cascade enable/disable autoload association on Find and FindAll query.
func (q Query) Cascade(c bool) Query {
	q.CascadeQuery = Cascade(c)
	return q
}

// Preload field association.
func (q Query) Preload(field string) Query {
	q.PreloadQuery = append(q.PreloadQuery, field)
	return q
}

// UsePrimary database.
func (q Query) UsePrimary() Query {
	q.UsePrimaryDb = true
	return q
}

// String describe query as string.
func (q Query) String() string {
	if q.SQLQuery.Statement != "" {
		return q.SQLQuery.String()
	}

	var builder strings.Builder
	builder.WriteString("rel")

	if q.UsePrimaryDb {
		builder.WriteString(".UsePrimary()")
	}

	if q.Table != "" {
		builder.WriteString(".From(\"")
		builder.WriteString(q.Table)
		builder.WriteString("\")")
	}

	if len(q.SelectQuery.Fields) != 0 {
		builder.WriteString(".Select(\"")
		builder.WriteString(strings.Join(q.SelectQuery.Fields, "\", \""))
		builder.WriteString("\")")
	}

	if q.SelectQuery.OnlyDistinct {
		builder.WriteString(".Distinct()")
	}

	for _, jq := range q.JoinQuery {
		builder.WriteString(".JoinWith(\"")
		builder.WriteString(jq.Mode)
		builder.WriteString("\", \"")
		builder.WriteString(jq.Table)
		builder.WriteString("\", \"")
		builder.WriteString(jq.From)
		builder.WriteString("\", \"")
		builder.WriteString(jq.To)
		builder.WriteString("\")")
	}

	if !q.WhereQuery.None() {
		builder.WriteString(".Where(")
		builder.WriteString(q.WhereQuery.String())
		builder.WriteByte(')')
	}

	if len(q.GroupQuery.Fields) != 0 {
		builder.WriteString(".Group(\"")
		builder.WriteString(strings.Join(q.GroupQuery.Fields, "\", \""))
		builder.WriteString("\")")

		if !q.GroupQuery.Filter.None() {
			builder.WriteString(".Having(")
			builder.WriteString(q.GroupQuery.Filter.String())
			builder.WriteByte(')')
		}
	}

	for _, sq := range q.SortQuery {
		if sq.Asc() {
			builder.WriteString(".SortAsc(\"")
		} else {
			builder.WriteString(".SortDesc(\"")
		}
		builder.WriteString(sq.Field)
		builder.WriteString("\")")
	}

	if q.LimitQuery > 0 {
		builder.WriteString(".Limit(")
		builder.WriteString(strconv.Itoa(int(q.LimitQuery)))
		builder.WriteString(")")
	}

	if q.OffsetQuery > 0 {
		builder.WriteString(".Offset(")
		builder.WriteString(strconv.Itoa(int(q.OffsetQuery)))
		builder.WriteString(")")
	}

	if q.LockQuery != "" {
		builder.WriteString(".Lock(\"")
		builder.WriteString(string(q.LockQuery))
		builder.WriteString("\")")
	}

	if q.UnscopedQuery {
		builder.WriteString(".Unscoped()")
	}

	if q.ReloadQuery {
		builder.WriteString(".Reload()")
	}

	if !q.CascadeQuery {
		builder.WriteString(".Cascade(false)")
	}

	if len(q.PreloadQuery) != 0 {
		builder.WriteString(".Preload(\"")
		builder.WriteString(strings.Join(q.PreloadQuery, "\", \""))
		builder.WriteString("\")")
	}

	if str := builder.String(); str != "rel" {
		return str
	}
	return ""
}

func newQuery() Query {
	return Query{
		CascadeQuery: true,
	}
}

// Select query create a query with chainable syntax, using select as the starting point.
func Select(fields ...string) Query {
	query := newQuery()
	query.SelectQuery.Fields = fields
	return query
}

// From create a query with chainable syntax, using from as the starting point.
func From(table string) Query {
	query := newQuery()
	query.Table = table
	return query
}

// Join create a query with chainable syntax, using join as the starting point.
func Join(table string, filter ...FilterQuery) Query {
	return JoinOn(table, "", "", filter...)
}

// JoinOn create a query with chainable syntax, using join as the starting point.
func JoinOn(table string, from string, to string, filter ...FilterQuery) Query {
	return JoinWith("JOIN", table, from, to, filter...)
}

// JoinWith create a query with chainable syntax, using join as the starting point.
func JoinWith(mode string, table string, from string, to string, filter ...FilterQuery) Query {
	query := newQuery()
	query.JoinQuery = []JoinQuery{
		NewJoinWith(mode, table, from, to, filter...),
	}
	return query
}

// Joinf create a query with chainable syntax, using join as the starting point.
func Joinf(expr string, args ...interface{}) Query {
	query := newQuery()
	query.JoinQuery = []JoinQuery{
		NewJoinFragment(expr, args...),
	}
	return query
}

// Where create a query with chainable syntax, using where as the starting point.
func Where(filters ...FilterQuery) Query {
	query := newQuery()
	query.WhereQuery = And(filters...)
	return query
}

func UsePrimary() Query {
	query := newQuery()
	query.UsePrimaryDb = true
	return query
}

// Offset  Query.
type Offset int

// Build query.
func (o Offset) Build(query *Query) {
	query.OffsetQuery = o
}

// Limit options.
// When passed as query, it limits returned result from database.
// When passed as column option, it sets the maximum size of the string/text/binary/integer columns.
type Limit int

// Build query.
func (l Limit) Build(query *Query) {
	query.LimitQuery = l
}

func (l Limit) applyColumn(column *Column) {
	column.Limit = int(l)
}

// Lock query.
// This query will be ignored if used outside of transaction.
type Lock string

// Build query.
func (l Lock) Build(query *Query) {
	query.LockQuery = l
}

// ForUpdate lock query.
func ForUpdate() Lock {
	return "FOR UPDATE"
}

// Unscoped query.
type Unscoped bool

// Build query.
func (u Unscoped) Build(query *Query) {
	query.UnscopedQuery = u
}

// Apply mutation.
func (u Unscoped) Apply(doc *Document, mutation *Mutation) {
	mutation.Unscoped = u
}

// Preload query.
type Preload string

// Build query.
func (p Preload) Build(query *Query) {
	query.PreloadQuery = append(query.PreloadQuery, string(p))
}
