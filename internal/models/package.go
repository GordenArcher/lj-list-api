package models

// PackageItem is a single item included in a fixed package bundle.
// ProductID is stored as a string so it can carry either a product UUID or a
// legacy numeric reference, depending on the client payload.
type PackageItem struct {
	ProductID string   `json:"product_id"`
	Qty       int      `json:"qty"`
	Label     string   `json:"label"`
	Emoji     string   `json:"emoji"`
	ImageURL  string   `json:"image_url,omitempty"`
	Product   *Product `json:"product,omitempty"`
}

// FixedPackage mirrors the storefront's package cards.
type FixedPackage struct {
	ID          string        `json:"id"`
	SortOrder   int           `json:"-"`
	Name        string        `json:"name"`
	Tagline     string        `json:"tagline"`
	Price       string        `json:"price"`
	Monthly     string        `json:"monthly"`
	Tag         string        `json:"tag"`
	Popular     bool          `json:"popular"`
	RiceOptions string        `json:"rice_options,omitempty"`
	Items       []PackageItem `json:"items"`
}

// SimplePackage is used for the Provisions and Detergents department bundles.
type SimplePackage struct {
	ID        string `json:"id"`
	SortOrder int    `json:"-"`
	Name      string `json:"name"`
	Price     int    `json:"price"`
	Items     string `json:"items"`
}

// PackageCatalog is the top-level response for the package endpoints.
type PackageCatalog struct {
	MinOrder           int             `json:"min_order"`
	PackageOptions     []string        `json:"package_options"`
	FixedPackages      []FixedPackage  `json:"fixed_packages"`
	ProvisionsPackages []SimplePackage `json:"provisions_packages"`
	DetergentPackages  []SimplePackage `json:"detergent_packages"`
}
