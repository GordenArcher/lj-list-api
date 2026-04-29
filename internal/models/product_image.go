package models

import "time"

// ProductImage is a single product gallery asset. Images are managed as their
// own resource so admins can upload many images per product and delete any one
// of them later using the image UUID.
type ProductImage struct {
	ID                 string    `json:"id"`
	ProductID          string    `json:"product_id"`
	ImageURL           string    `json:"image_url"`
	CloudinaryPublicID string    `json:"-"`
	CreatedAt          time.Time `json:"created_at"`
}
