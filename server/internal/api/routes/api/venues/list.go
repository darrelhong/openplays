package venues

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"openplays/server/internal/db"
)

type ListBody struct {
	Items []VenuePublic `json:"items"`
}

type ListOutput struct {
	Body ListBody
}

func RegisterList(api huma.API, queries *db.Queries) {
	huma.Register(api, huma.Operation{
		OperationID: "list-venues",
		Summary:     "List venues with postal codes",
		Description: "Returns all resolved venues that have a postal code and coordinates. Useful for building venue selectors.",
		Method:      http.MethodGet,
		Path:        "/",
		Tags:        []string{"Venues"},
	}, func(ctx context.Context, input *struct{}) (*ListOutput, error) {
		rows, err := queries.ListVenuesWithPostalCode(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to list venues", err)
		}

		items := make([]VenuePublic, len(rows))
		for i, r := range rows {
			items[i] = VenuePublic{
				ID:         r.ID,
				Name:       r.Name,
				PostalCode: *r.PostalCode, // safe: query filters WHERE postal_code IS NOT NULL
				Latitude:   r.Latitude,
				Longitude:  r.Longitude,
			}
		}

		return &ListOutput{Body: ListBody{Items: items}}, nil
	})
}
