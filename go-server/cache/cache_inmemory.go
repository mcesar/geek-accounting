// +build inmemory

package cache

import (
	"reflect"
)

type inMemoryCache struct{}

var cache = map[string]interface{}{}

func NewInMemoryCache() Cache {
	return inMemoryCache{}
}

func (c inMemoryCache) Get(key string, item interface{}) error {
	if v, ok := cache[key]; ok {
		reflect.Indirect(reflect.ValueOf(item)).Set(reflect.ValueOf(v))
	}
	return nil
}

func (c inMemoryCache) Set(key string, item interface{}) error {
	cache[key] = item
	return nil
}

func (c inMemoryCache) Delete(key string) error {
	delete(cache, key)
	return nil
}

func (c inMemoryCache) Flush() error {
	cache = map[string]interface{}{}
	return nil
}
