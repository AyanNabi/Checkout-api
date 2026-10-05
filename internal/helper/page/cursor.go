package page

import (
	"encoding/base64"
	"encoding/json"
)

type Cursor struct {
	ID int `json:"id"`
}

func EncodeCursor(id int) (string, error) {
	cursor := Cursor{
		ID: id,
	}

	data, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

func DecodeCursor(value string) (Cursor, error) {
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return Cursor{}, err
	}

	var cursor Cursor

	if err := json.Unmarshal(data, &cursor); err != nil {
		return Cursor{}, err
	}

	return cursor, nil
}
