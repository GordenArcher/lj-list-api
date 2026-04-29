package models

import "time"

// CartItem is a snapshot of a product at the moment the application was
// submitted. We store name, price, and image_url alongside the product_id
// so historical applications remain accurate even if the product catalog
// changes. Subtotal is price * quantity, precomputed so the frontend and
// admin dashboard don't recalculate it on every render.
type CartItem struct {
	ProductID string `json:"product_id"`
	Name      string `json:"name"`
	ImageURL  string `json:"image_url"`
	Price     int    `json:"price"`
	Quantity  int    `json:"quantity"`
	Subtotal  int    `json:"subtotal"`
}

// ApplicationCustomer is the lightweight customer profile attached to admin
// application responses. It intentionally excludes password metadata and other
// account internals while still giving the dashboard enough context to identify
// who submitted the application.
type ApplicationCustomer struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	DisplayName string  `json:"display_name"`
	Phone       *string `json:"phone,omitempty"`
	Role        string  `json:"role"`
}

// Application represents a customer's grocery order request. PackageType
// is either "fixed" or "custom". For fixed packages, PackageName holds the
// selected tier and CartItems is empty. For custom packages, CartItems
// contains the line items and PackageName is empty.
type Application struct {
	ID              string               `json:"id"`
	UserID          string               `json:"user_id"`
	Customer        *ApplicationCustomer `json:"customer,omitempty"`
	PackageType     string               `json:"package_type"` // "fixed" or "custom"
	PackageName     string               `json:"package_name,omitempty"`
	CartItems       []CartItem           `json:"cart_items,omitempty"`
	TotalAmount     int                  `json:"total_amount"`
	MonthlyAmount   int                  `json:"monthly_amount"`
	Status          string               `json:"status"` // pending, reviewed, approved, declined
	StaffNumber     string               `json:"staff_number"`
	MandateNumber   string               `json:"mandate_number"`
	Institution     string               `json:"institution"`
	GhanaCardNumber string               `json:"ghana_card_number"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
}
