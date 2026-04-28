package models

// Product is a single item in the grocery catalog. Price is an integer
// representing Ghana cedis, no pesewa precision needed for this domain.
// ImageURL points to Cloudinary; the storage package handles uploads, the
// model just holds the reference.
type Product struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Price    int    `json:"price"`
	ImageURL string `json:"image_url"`
	Unit     string `json:"unit"` // "bag", "bottle", "carton", "pack", "tin", "box"
	Active   bool   `json:"active"`
}
