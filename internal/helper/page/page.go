package page

type Page[T any] struct {
	Data       []T     `json:"data"`
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

func CreatePage[T any](
	data []T,
	limit int,
	getID func(T) int,
) Page[T] {

	hasMore := len(data) > limit

	if hasMore {
		data = data[:limit]
	}

	var nextCursor *string

	if hasMore && len(data) > 0 {
		id := getID(data[len(data)-1])

		cursor, err := EncodeCursor(id)
		if err != nil {
			return Page[T]{}
		}

		nextCursor = &cursor
	}

	return Page[T]{
		Data:       data,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}
}
