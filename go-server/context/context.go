package context

import (
	"github.com/mcesarhm/geek-accounting/go-server/cache"
	"github.com/mcesarhm/geek-accounting/go-server/db"
)

type Context struct {
	Db    db.Db
	Cache cache.Cache
}
