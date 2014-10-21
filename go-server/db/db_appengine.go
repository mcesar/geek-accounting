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

type CKey struct{ DsKey *datastore.Key }

type CKeys []*datastore.Key

func (key CKey) String() string {
	if key.DsKey == nil {
		return ""
	}
	return key.DsKey.String()
}

func (key CKey) Encode() string {
	return key.DsKey.Encode()
}

func (key CKey) Parent() Key {
	return CKey{key.DsKey.Parent()}
}

func (key CKey) IsZero() bool {
	return key.DsKey == nil
}

func (key CKey) MarshalJSON() ([]byte, error) {
	s := ""
	if key.DsKey != nil {
		s = key.DsKey.Encode()
	}
	return []byte(fmt.Sprintf("\"%v\"", s)), nil
}

func (key CKey) UnmarshalJSON(b []byte) error {
	k, err := datastore.DecodeKey(string(b)[1 : len(string(b))-1])
	if err != nil {
		return err
	}
	key.DsKey = k
	return nil
}

func (keys CKeys) Len() int {
	return len(keys)
}

func (keys CKeys) KeyAt(i int) Key {
	return CKey{keys[i]}
}

func (keys CKeys) Append(key Key) Keys {
	return Keys(append(keys, key.(CKey).DsKey))
}

func (keys CKeys) Swap(i, j int) {
	keys[i], keys[j] = keys[j], keys[i]
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
	identifier.SetKey(CKey{key})
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
			identifier.SetKey(CKey{keys[i]})
		}
	}
	return CKeys(keys), items, err
}

func (db AppengineDb) GetAllFromCache(kind string, ancestor Key, items interface{}, filteredItems interface{}, filters M, order []string, cacheKey string) (Keys, interface{}, error) {
	var arr []interface{}
	var keys []*datastore.Key
	_, err := memcache.Gob.Get(db.c, cacheKey, &arr)
	if err == memcache.ErrCacheMiss {
		q := datastore.NewQuery(kind)
		if ancestor.(CKey).DsKey != nil {
			q = q.Ancestor(ancestor.(CKey).DsKey)
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
			keysChunk := make(CKeys, chunkSize)
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
			if chunkKeys, filteredItems, err = filter(db, chunkKeys, arr[1], filters, filteredItems); err != nil {
				return nil, nil, err
			}
		}
		for i := 0; i < chunkKeys.Len(); i++ {
			keys = append(keys, chunkKeys.KeyAt(i).(CKey).DsKey)
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
		sort.Sort(ByFields{CKeys(keys), reflect.ValueOf(items), order})
	}

	return CKeys(keys), items, nil
}

func (db AppengineDb) Save(item interface{}, kind string, ancestor string, param map[string]string) (key Key, err error) {
	if _, ok := item.(ValidationMessager); ok {
		vm := item.(ValidationMessager).ValidationMessage(db, param)
		if len(vm) > 0 {
			return CKey{}, errors.New(vm)
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
	var ckey CKey
	if !identifier.GetKey().IsZero() {
		ckey = identifier.GetKey().(CKey)
	} else {
		ckey = CKey{datastore.NewIncompleteKey(db.c, kind, ancestorKey)}
	}
	ckey.DsKey, err = datastore.Put(db.c, ckey.DsKey, item)
	if err != nil {
		return
	}
	if !ckey.IsZero() {
		identifier.SetKey(ckey)
		key = ckey
	}

	return
}

func (db AppengineDb) Delete(key Key) error {
	return datastore.Delete(db.c, key.(CKey).DsKey)
}

func (db AppengineDb) Execute(f func() error) error {
	return datastore.RunInTransaction(db.c, func(appengine.Context) (err error) {
		return f()
	}, nil)
}

func (db AppengineDb) DecodeKey(keyAsString string) (Key, error) {
	k, err := datastore.DecodeKey(keyAsString)
	return CKey{k}, err
}

func (db AppengineDb) NewStringKey(kind, key string) Key {
	return CKey{datastore.NewKey(db.c, kind, key, 0, nil)}
}

func (db AppengineDb) NewKey() Key {
	return CKey{nil}
}

func (db AppengineDb) NewKeys() Keys {
	return CKeys{}
}
