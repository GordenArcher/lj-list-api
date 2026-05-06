package models

// Category is a storefront category row used to validate and filter products.
type Category struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"-"`
	Active    bool   `json:"-"`
}
