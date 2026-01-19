package models

type Field struct {
	ID          string      `json:"id"`
	Key         string      `json:"key"`
	Name        string      `json:"name"`
	Custom      bool        `json:"custom"`
	Orderable   bool        `json:"orderable"`
	Navigable   bool        `json:"navigable"`
	Searchable  bool        `json:"searchable"`
	ClauseNames []string    `json:"clauseNames"`
	Schema      FieldSchema `json:"schema,omitempty"`
}

type FieldSchema struct {
	Type     string `json:"type"`
	Items    string `json:"items,omitempty"`
	System   string `json:"system,omitempty"`
	Custom   string `json:"custom,omitempty"`
	CustomID int    `json:"customId,omitempty"`
}
