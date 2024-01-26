package plugin

import (
	"context"

	"github.com/rockset/rockset-go-client/openapi"
	"github.com/rockset/rockset-go-client/option"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate -o fake . Queryer
type Queryer interface {
	Query(context.Context, string, ...option.QueryOption) (openapi.QueryResponse, error)
}
