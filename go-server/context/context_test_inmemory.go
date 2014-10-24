// +build inmemory
// +build test

package context

import (
	"github.com/mcesarhm/geek-accounting/go-server/cache"
	"github.com/mcesarhm/geek-accounting/go-server/db"
)

type inMemoryContext struct {
}

func (_ inMemoryContext) Close() {
}

func NewContext(c *Context) (inMemoryContext, error) {
	c.Db = db.NewInMemoryDb()
	c.Cache = cache.NewInMemoryCache()
	c.Cache.Flush()
	return inMemoryContext{}, nil
}
