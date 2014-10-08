// +build appengine

package cache

import (
	"appengine"
	"appengine/memcache"
)

type AppengineCache struct{ c appengine.Context }

func NewAppengineCache(c appengine.Context) Cache {
	return AppengineCache{c}
}

func (ac AppengineCache) Get(key string, item interface{}) error {
	if _, err := memcache.Gob.Get(ac.c, key, item); err != nil && err != memcache.ErrCacheMiss {
		return err
	}
	return nil
}

func (ac AppengineCache) Set(key string, item interface{}) error {
	cacheItem := &memcache.Item{
		Key:    key,
		Object: item,
	}
	return memcache.Gob.Set(ac.c, cacheItem)
}

func (ac AppengineCache) Delete(key string) error {
	if err := memcache.Delete(ac.c, key); err != nil && err != memcache.ErrCacheMiss {
		return err
	}
	return nil
}

func (ac AppengineCache) Flush() error {
	return memcache.Flush(ac.c)
}
