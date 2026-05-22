package cms

import "encoding/json"

type UpdateSectionsRequest struct {
	Sections json.RawMessage `json:"sections"`
}

type UpdateVisibilityRequest struct {
	IsPublished bool `json:"is_published"`
}
