// +build appengine AND test

package context

import (
	"appengine/aetest"
	"github.com/mcesarhm/geek-accounting/go-server/cache"
	"github.com/mcesarhm/geek-accounting/go-server/db"
)

func NewContext(c *Context) (aetest.Context, error) {
	ac, err := aetest.NewContext(nil)
	if err != nil {
		return nil, err
	}
	c.Db = db.NewAppengineDb(ac)
	c.Cache = cache.NewAppengineCache(ac)
	return ac, nil
}
