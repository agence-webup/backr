package backr

import "context"

type API interface {
	Listen(ctx context.Context) error
}
