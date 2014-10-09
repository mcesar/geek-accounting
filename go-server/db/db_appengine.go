// +build appengine

package db

import (
	"appengine"
	"appengine/datastore"
	"appengine/memcache"
	"errors"
	"fmt"
	"github.com/mcesarhm/geek-accounting/go-server/util"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type AppengineDb struct{ c appengine.Context }

func NewAppengineDb(c appengine.Context) Db {
	return AppengineDb{c}
}

type Key struct{ DsKey *datastore.Key }

type Keys []*datastore.Key

func NewKey() Key {
	return Key{nil}
}

func (key Key) String() string {
	if key.DsKey == nil {
		return ""
	}
	return key.DsKey.String()
}

func (key Key) Encode() string {
	return key.DsKey.Encode()
}

func (key Key) Parent() Key {
	return Key{key.DsKey.Parent()}
}

func (key Key) IsZero() bool {
	return key.DsKey == nil
}

func (key Key) MarshalJSON() ([]byte, error) {
	s := ""
	if key.DsKey != nil {
		s = key.DsKey.Encode()
	}
	return []byte(fmt.Sprintf("\"%v\"", s)), nil
}

func (key Key) UnmarshalJSON(b []byte) error {
	k, err := datastore.DecodeKey(string(b)[1 : len(string(b))-1])
	if err != nil {
		return err
	}
	key.DsKey = k
	return nil
}

func (keys Keys) KeyAt(i int) Key {
	return Key{keys[i]}
}

func (keys Keys) Append(key Key) Keys {
	return Keys(append(keys, key.DsKey))
}

func (db AppengineDb) Get(item interface{}, keyAsString string) (result interface{}, err error) {
	key, err := datastore.DecodeKey(keyAsString)
	if err != nil {
		return
	}
	err = datastore.Get(db.c, key, item)
	if err != nil {
		return
	}
	identifier := item.(Identifier)
	identifier.SetKey(Key{key})
	result = item
	return
}

func (db AppengineDb) GetAll(kind string, ancestor string, items interface{}, filters M, orderKeys []string) (Keys, interface{}, error) {
	return db.GetAllWithLimit(kind, ancestor, items, filters, orderKeys, 0)
}

func (db AppengineDb) GetAllWithLimit(kind string, ancestor string, items interface{}, filters M, orderKeys []string, limit int) (Keys, interface{}, error) {
	q := datastore.NewQuery(kind)
	if len(ancestor) > 0 {
		ancestorKey, err := datastore.DecodeKey(ancestor)
		if err != nil {
			return nil, nil, err
		}
		q = q.Ancestor(ancestorKey)
	}
	if orderKeys != nil {
		for _, o := range orderKeys {
			q = q.Order(o)
		}
	}
	if filters != nil {
		for k, v := range filters {
			q = q.Filter(k, v)
		}
	}
	if items == nil {
		q = q.KeysOnly()
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	keys, err := q.GetAll(db.c, items)
	if items != nil {
		v := reflect.ValueOf(items).Elem()
		for i := 0; i < v.Len(); i++ {
			ptr := v.Index(i)
			if ptr.Kind() != reflect.Ptr {
				ptr = ptr.Addr()
			}
			identifier := ptr.Interface().(Identifier)
			identifier.SetKey(Key{keys[i]})
		}
	}
	return keys, items, err
}

func (db AppengineDb) GetAllFromCache(kind string, ancestor Key, items interface{}, filteredItems interface{}, filters M, order []string, cacheKey string) (Keys, interface{}, error) {
	var arr []interface{}
	var keys []*datastore.Key
	_, err := memcache.Gob.Get(db.c, cacheKey, &arr)
	if err == memcache.ErrCacheMiss {
		q := datastore.NewQuery(kind)
		if ancestor.DsKey != nil {
			q = q.Ancestor(ancestor.DsKey)
		}
		if keys, err = q.GetAll(db.c, items); err != nil {
			return nil, nil, err
		}
		itemsValue := reflect.Indirect(reflect.ValueOf(items))
		items = itemsValue.Interface()
		nextKey := cacheKey
		nextChunk := 0
		i := 0
		for {
			chunkSize := util.Min(MaxItemsPerMemcacheEntry, len(keys)-i)
			keysChunk := make(Keys, chunkSize)
			itemsChunk := reflect.MakeSlice(itemsValue.Type(), chunkSize, chunkSize)
			for j := 0; j < chunkSize; j++ {
				keysChunk[j] = keys[i+j]
				itemsChunk.Index(j).Set(itemsValue.Index(i + j))
			}
			if i+chunkSize >= len(keys) {
				nextKey = ""
			} else {
				nextKey = strings.Split(cacheKey, "|")[0] + "|" + strconv.Itoa(nextChunk+1)
			}
			chunkArr := []interface{}{keysAsStrings(keysChunk), itemsChunk.Interface(), nextKey}
			memcacheItem := &memcache.Item{
				Key:    cacheKey,
				Object: chunkArr,
			}
			if err = memcache.Gob.Set(db.c, memcacheItem); err != nil {
				return nil, nil, err
			}
			if nextChunk == 0 {
				arr = chunkArr
			}
			if len(nextKey) == 0 {
				break
			}
			nextChunk++
			cacheKey = strings.Split(cacheKey, "|")[0] + "|" + strconv.Itoa(nextChunk)
			i += MaxItemsPerMemcacheEntry
		}
	} else if err != nil {
		return nil, nil, err
	}
	keys = []*datastore.Key{}
	for {
		chunkKeys, err := stringsAsKeys(db, arr[0].([]string))
		if err != nil {
			return nil, nil, err
		}
		if filters == nil {
			itemsValue := reflect.ValueOf(arr[1])
			resultItems := reflect.ValueOf(filteredItems)
			for i := 0; i < itemsValue.Len(); i++ {
				resultItems = reflect.Append(resultItems, itemsValue.Index(i))
			}
			filteredItems = resultItems.Interface()
		} else {
			if chunkKeys, filteredItems, err = filter(chunkKeys, arr[1], filters, filteredItems); err != nil {
				return nil, nil, err
			}
		}
		for i := 0; i < len(chunkKeys); i++ {
			keys = append(keys, chunkKeys[i])
		}
		if len(arr[2].(string)) == 0 {
			break
		}
		if _, err := memcache.Gob.Get(db.c, arr[2].(string), &arr); err != nil {
			return nil, nil, err
		}
	}
	items = filteredItems
	if order != nil {
		sort.Sort(ByFields{keys, reflect.ValueOf(items), order})
	}

	return keys, items, nil
}

func (db AppengineDb) Save(item interface{}, kind string, ancestor string, param map[string]string) (key Key, err error) {
	if _, ok := item.(ValidationMessager); ok {
		vm := item.(ValidationMessager).ValidationMessage(db, param)
		if len(vm) > 0 {
			return Key{}, errors.New(vm)
		}
	}
	var ancestorKey *datastore.Key
	if len(ancestor) > 0 {
		ancestorKey, err = datastore.DecodeKey(ancestor)
		if err != nil {
			return
		}
	}
	identifier := item.(Identifier)
	if !identifier.GetKey().IsZero() {
		key = identifier.GetKey()
	} else {
		key = Key{datastore.NewIncompleteKey(db.c, kind, ancestorKey)}
	}
	key.DsKey, err = datastore.Put(db.c, key.DsKey, item)
	if err != nil {
		return
	}
	if !key.IsZero() {
		identifier.SetKey(key)
	}

	return
}

func (db AppengineDb) Delete(key Key) error {
	return datastore.Delete(db.c, key.DsKey)
}

func (db AppengineDb) Execute(f func() error) error {
	return datastore.RunInTransaction(db.c, func(appengine.Context) (err error) {
		return f()
	}, nil)
}

func (db AppengineDb) DecodeKey(keyAsString string) (Key, error) {
	k, err := datastore.DecodeKey(keyAsString)
	return Key{k}, err
}

func (db AppengineDb) NewStringKey(kind, key string) Key {
	return Key{datastore.NewKey(db.c, kind, key, 0, nil)}
}
