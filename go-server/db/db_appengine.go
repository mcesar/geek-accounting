// +build appengine

package db

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/mcesarhm/geek-accounting/go-server/cache"
	xmath "github.com/mcesarhm/geek-accounting/go-server/extensions/math"

	"appengine"
	"appengine/datastore"
)

type appengineDb struct{ c appengine.Context }

func NewAppengineDb(c appengine.Context) Db {
	return appengineDb{c}
}

type CKey struct{ DsKey *datastore.Key }

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

func (db appengineDb) Get(item interface{}, keyAsString string) (result interface{}, err error) {
	key, err := datastore.DecodeKey(keyAsString)
	if err != nil {
		return
	}
	err = datastore.Get(db.c, key, item)
	if err != nil {
		logStackTrace(db.c, err)
		return
	}
	identifier := item.(Identifier)
	identifier.SetKey(CKey{key})
	result = item
	return
}

func (db appengineDb) GetAll(kind string, ancestor string, items interface{}, filters M, orderKeys []string) (Keys, interface{}, error) {
	return db.GetAllWithLimit(kind, ancestor, items, filters, orderKeys, 0)
}

func (db appengineDb) GetAllWithLimit(kind string, ancestor string, items interface{}, filters M, orderKeys []string, limit int) (Keys, interface{}, error) {
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
			if _, ok := v.(CKey); ok {
				v = v.(CKey).DsKey
			}
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
	if err != nil {
		logStackTrace(db.c, err)
	} else {
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
	}
	return toKeys(keys), items, err
}

func (db appengineDb) GetAllFromCache(kind string, ancestor string, items interface{}, filters M, order []string, c cache.Cache, cacheKey string) (Keys, interface{}, error) {
	arr := []interface{}{}
	err := c.Get(cacheKey, &arr)
	if err == nil && len(arr) == 0 {
		q := datastore.NewQuery(kind)
		if len(ancestor) > 0 {
			ancestorKey, err := datastore.DecodeKey(ancestor)
			if err != nil {
				return nil, nil, err
			}
			q = q.Ancestor(ancestorKey)
		}
		var keys []*datastore.Key
		if keys, err = q.GetAll(db.c, items); err != nil {
			logStackTrace(db.c, err)
			return nil, nil, err
		}
		itemsValue := reflect.Indirect(reflect.ValueOf(items))
		nextKey := cacheKey
		nextChunk := 0
		i := 0
		for {
			chunkSize := xmath.Min(MaxItemsPerMemcacheEntry, len(keys)-i)
			keysChunk := make(Keys, chunkSize)
			itemsChunk := reflect.MakeSlice(itemsValue.Type(), chunkSize, chunkSize)
			for j := 0; j < chunkSize; j++ {
				keysChunk[j] = CKey{keys[i+j]}
				itemsChunk.Index(j).Set(itemsValue.Index(i + j))
			}
			if i+chunkSize >= len(keys) {
				nextKey = ""
			} else {
				nextKey = strings.Split(cacheKey, "|")[0] + "|" + strconv.Itoa(nextChunk+1)
			}
			chunkArr := []interface{}{keysAsStrings(keysChunk), itemsChunk.Interface(), nextKey}
			if err = c.Set(cacheKey, chunkArr); err != nil {
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
	keys := Keys{}
	resultItems := reflect.MakeSlice(reflect.ValueOf(items).Elem().Type(), 0, 0)
	for {
		chunkKeys, err := stringsAsKeys(db, arr[0].([]string))
		if err != nil {
			return nil, nil, err
		}
		var chunkItemsValue reflect.Value
		if filters == nil {
			chunkItemsValue = reflect.ValueOf(arr[1])
		} else {
			var filteredItems interface{}
			if chunkKeys, filteredItems, err = filter(db, chunkKeys, arr[1], filters); err != nil {
				return nil, nil, err
			}
			chunkItemsValue = reflect.ValueOf(filteredItems)
		}
		for i := 0; i < chunkKeys.Len(); i++ {
			keys = keys.Append(chunkKeys.KeyAt(i))
		}
		for i := 0; i < chunkItemsValue.Len(); i++ {
			resultItems = reflect.Append(resultItems, chunkItemsValue.Index(i))
		}
		if len(arr[2].(string)) == 0 {
			break
		}
		if err := c.Get(arr[2].(string), &arr); err != nil {
			logStackTrace(db.c, err)
			return nil, nil, err
		}
	}
	if order != nil {
		sort.Sort(byFields{keys, resultItems, order})
	}
	reflect.Indirect(reflect.ValueOf(items)).Set(resultItems)

	return keys, items, nil
}

func (db appengineDb) Save(item interface{}, kind string, ancestor string, param map[string]string) (key Key, err error) {
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

func (db appengineDb) Delete(key Key) error {
	return datastore.Delete(db.c, key.(CKey).DsKey)
}

func (db appengineDb) Execute(f func(Db) error) error {
	return datastore.RunInTransaction(db.c, func(tc appengine.Context) (err error) {
		return f(NewAppengineDb(tc))
	}, nil)
}

func (db appengineDb) DecodeKey(keyAsString string) (Key, error) {
	k, err := datastore.DecodeKey(keyAsString)
	return CKey{k}, err
}

func (db appengineDb) NewStringKey(kind, key string) Key {
	return CKey{datastore.NewKey(db.c, kind, key, 0, nil)}
}

func (db appengineDb) NewKey() Key {
	return CKey{nil}
}

func (db appengineDb) Log(s string) {
	db.c.Infof(s)
}

func toKeys(dsKeys []*datastore.Key) Keys {
	result := Keys{}
	for _, k := range dsKeys {
		result = result.Append(CKey{k})
	}
	return result
}

func logStackTrace(c appengine.Context, err error) {
	var stack [4096]byte
	runtime.Stack(stack[:], false)
	c.Errorf("%q\n%s\n", err, stack[:])
}
