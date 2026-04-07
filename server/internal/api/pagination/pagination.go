// Package pagination provides reusable cursor-based pagination types
// for API responses.
//
// Uses forward-only cursor pagination with an opaque string cursor.
// The server requests page_size + 1 rows and uses the extra row to
// determine if there are more pages, without needing a separate COUNT query.
package pagination

// Page is a generic paginated response wrapper.
type Page[T any] struct {
	Items      []T     `json:"items"`
	Total      int64   `json:"total" doc:"Total number of matching results across all pages"`
	NextCursor *string `json:"next_cursor,omitempty" doc:"Opaque cursor; pass as cursor to get the next page"`
	HasMore    bool    `json:"has_more" doc:"Whether there are more results after this page"`
}

// Paginate takes a slice of rows (fetched with limit + 1), the
// requested page size, the total count, and a function to extract
// the cursor string from the last item.
func Paginate[T any](rows []T, pageSize int64, total int64, getCursor func(T) string) Page[T] {
	hasMore := int64(len(rows)) > pageSize
	if hasMore {
		rows = rows[:pageSize]
	}

	var nextCursor *string
	if hasMore && len(rows) > 0 {
		c := getCursor(rows[len(rows)-1])
		nextCursor = &c
	}

	return Page[T]{
		Items:      rows,
		Total:      total,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}
}
