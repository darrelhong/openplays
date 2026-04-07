// Package pagination provides reusable cursor-based pagination types
// for API responses.
//
// Uses forward-only cursor pagination: the client passes the last seen
// item ID as the cursor to get the next page. The server requests
// page_size + 1 rows and uses the extra row to determine if there are
// more pages, without needing a separate COUNT query.
package pagination

// Page is a generic paginated response wrapper.
type Page[T any] struct {
	Items      []T    `json:"items"`
	Total      int64  `json:"total" doc:"Total number of matching results across all pages"`
	NextCursor *int64 `json:"next_cursor,omitempty" doc:"ID of the last item; pass as cursor to get the next page"`
	HasMore    bool   `json:"has_more" doc:"Whether there are more results after this page"`
}

// Paginate takes a slice of rows (fetched with limit + 1), the
// requested page size, the total count, and a function to extract
// the cursor ID from each item.
func Paginate[T any](rows []T, pageSize int64, total int64, getID func(T) int64) Page[T] {
	hasMore := int64(len(rows)) > pageSize
	if hasMore {
		rows = rows[:pageSize]
	}

	var nextCursor *int64
	if hasMore && len(rows) > 0 {
		id := getID(rows[len(rows)-1])
		nextCursor = &id
	}

	return Page[T]{
		Items:      rows,
		Total:      total,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}
}
