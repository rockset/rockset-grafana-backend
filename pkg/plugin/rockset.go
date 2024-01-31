package plugin

import (
	"context"

	"github.com/rockset/rockset-go-client"
	"github.com/rockset/rockset-go-client/openapi"
	"github.com/rockset/rockset-go-client/option"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate -o fake . RockClient
type RockClient interface {
	GetOrganization(context.Context) (openapi.Organization, error)
	Query(context.Context, string, ...option.QueryOption) (openapi.QueryResponse, error)
}

func RockFactory(options ...rockset.RockOption) (RockClient, error) {
	return rockset.NewClient(options...)
}
