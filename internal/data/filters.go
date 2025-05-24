package data

import (
	"slices"
	"strings"

	"github.com/Bekian/greenlight/internal/validator"
)

// struct to hold and set filters for query validation
// DIFF Note: the casing on the SortSafeList field has a lowercase "l" in the book for "list"; it is "SortSafelist" in the book.
type Filters struct {
	Page         int
	PageSize     int
	Sort         string
	SortSafeList []string // acceptable string values for sorting
}

// function to ensure the queried column is valid.
// DIFF Note: the original code uses a for loop to see if the requested sort is
// contained inside the SortSafeList
func (f Filters) sortColumn() string {
	if slices.Contains(f.SortSafeList, f.Sort) {
		return strings.TrimPrefix(f.Sort, "-")
	}

	// the queried sort column should've been checked before this
	// this should be unreachable, but this will prevent sql injection here.
	panic("unsafe sort parameter:" + f.Sort)
}

// return sort direction depending on prefix character, if any
func (f Filters) sortDirection() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}
	return "ASC"
}

// runs validation checks on the filter's values
func ValidateFilters(v *validator.Validator, f Filters) {
	// ensure page and page size fields are acceptable values
	// they have been pasted from the book to ensure functionality
	v.Check(f.Page > 0, "page", "must be greater than zero")
	v.Check(f.Page <= 10_000_000, "page", "must be a maximum of 10 million")
	v.Check(f.PageSize > 0, "page_size", "must be greater than zero")
	v.Check(f.PageSize <= 100, "page_size", "must be a maximum of 100")
	// validate sort parameter
	v.Check(validator.PermittedValue(f.Sort, f.SortSafeList...), "sort", "invalid sort value")
}
