package types

// SortDirection represents the direction for sorting
type SortDirection string

const (
	ASC  SortDirection = "ASC"
	DESC SortDirection = "DESC"
)

// SortParams represents sorting parameters
type SortParams struct {
	Sort      string `json:"sort" form:"sort" query:"sort"`
	Direction string `json:"direction" form:"direction" query:"direction"`
}
