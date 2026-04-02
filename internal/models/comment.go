package models

type Comment struct {
	ID           string    `json:"id"`
	Self         string    `json:"self"`
	Body         *ADF      `json:"body,omitempty"`
	Author       *User     `json:"author,omitempty"`
	Created      *JiraTime `json:"created,omitempty"`
	Updated      *JiraTime `json:"updated,omitempty"`
	UpdateAuthor *User     `json:"updateAuthor,omitempty"`
}

type CommentList struct {
	StartAt    int       `json:"startAt"`
	MaxResults int       `json:"maxResults"`
	Total      int       `json:"total"`
	Comments   []Comment `json:"comments"`
}
