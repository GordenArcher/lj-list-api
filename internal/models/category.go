package models

// Category is a storefront category row used to validate and filter products.
type Category struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description,omitempty"`
	Instructions    string `json:"instructions,omitempty"`
	Tag             string `json:"tag,omitempty"`
	RequiresInquiry bool   `json:"requires_inquiry"`
	Orderable       bool   `json:"orderable"`
	SortOrder       int    `json:"-"`
	Active          bool   `json:"active"`
}
