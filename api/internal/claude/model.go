package claude

import "errors"

var (
	ErrPageNotFound    = errors.New("page not found")
	ErrSectionNotFound = errors.New("section not found on page")
)

type GenerateRequest struct {
	PageSlug    string `json:"page_slug"    validate:"required"`
	SectionType string `json:"section_type" validate:"required"`
	Instruction string `json:"instruction"  validate:"required"`
}

type FieldChange struct {
	Field    string `json:"field"`
	Current  string `json:"current"`
	Proposed string `json:"proposed"`
	Notes    string `json:"notes"`
}

type GenerateResponse struct {
	Changes []FieldChange `json:"changes"`
}
