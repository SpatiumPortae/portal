package models

type File struct {
	Name  string `json:"name"`
	Bytes int64  `json:"bytes"`
}
