package models

// Product is a single item in the grocery catalog. Price is an integer
// representing Ghana cedis, no pesewa precision needed for this domain.
// ImageURL is the primary/compatibility image used by older clients.
// Images contains the full gallery so product media is no longer limited
// to a single asset.
type Product struct {
	ID              string         `json:"id"`
	LegacyID        *int           `json:"legacy_id,omitempty"`
	CategoryID      string         `json:"category_id"`
	Name            string         `json:"name"`
	Category        string         `json:"category"`
	Price           int            `json:"price"`
	OldPrice        *int           `json:"old_price,omitempty"`
	Tag             string         `json:"tag,omitempty"`
	DisplayTag      string         `json:"display_tag,omitempty"`
	Description     string         `json:"description,omitempty"`
	Instructions    string         `json:"instructions,omitempty"`
	RequiresInquiry bool           `json:"requires_inquiry"`
	Orderable       bool           `json:"orderable"`
	ImageURL        string         `json:"image_url"`
	Images          []ProductImage `json:"images,omitempty"`
	Unit            string         `json:"unit"` // "bag", "bottle", "carton", "pack", "tin", "box"
	Active          bool           `json:"active"`
}
