package explore

import (
	"errors"
	"sync"

	"k8s.io/client-go/openapi"
)

type cachedOpenAPIClient struct {
	pool *sync.Pool
}

func newCachedOpenAPIClient(c openapi.Client) (openapi.Client, error) {
	paths, err := c.Paths()
	if err != nil {
		return nil, err
	}
	p := &sync.Pool{
		New: func() interface{} {
			return paths
		},
	}
	return &cachedOpenAPIClient{
		pool: p,
	}, nil
}

func (c *cachedOpenAPIClient) Paths() (map[string]openapi.GroupVersion, error) {
	paths := c.pool.Get().(map[string]openapi.GroupVersion)
	defer c.pool.Put(paths)
	if len(paths) == 0 {
		return nil, errors.New("cached paths are empty")
	}
	return paths, nil
}
