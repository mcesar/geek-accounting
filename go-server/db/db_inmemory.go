// +build inmemory

package db

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/mcesarhm/geek-accounting/go-server/cache"
	"github.com/mcesarhm/geek-accounting/go-server/db"
	xmath "github.com/mcesarhm/geek-accounting/go-server/extensions/math"
)

type inMemoryDb struct {
	data map[string]*kindItems
}

type kindItems struct {
	items  map[int]interface{}
	lastId int
}

func NewInMemoryDb() Db {
	return inMemoryDb{map[string]*kindItems{}}
}

type CKey struct {
	id     int
	parent *CKey
	kind   string
}

func (key CKey) String() string {
	return key.stringSep("-")
}

func (key CKey) Encode() string {
	return key.String()
}

func (key CKey) Parent() Key {
	if key.parent == nil {
		return CKey{}
	}
	return *key.parent
}

func (key CKey) IsZero() bool {
	return key.id == 0 && key.parent == nil && key.kind == ""
}

func (key CKey) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%v\"", key.String())), nil
}

func (key CKey) UnmarshalJSON(b []byte) (err error) {
	return key.unmarshalJSONSep([]byte(b[1:len(string(b))-1]), "-")
}

func (key *CKey) unmarshalJSONSep(b []byte, separator string) (err error) {
	arr := strings.Split(string(b), separator)
	if key.id, err = strconv.Atoi(arr[0]); err != nil {
		return err
	}
	if len(arr[1]) > 0 {
		key.parent = &CKey{}
		if err := key.parent.unmarshalJSONSep([]byte(arr[1]), "."); err != nil {
			return err
		}
	} else {
		key.parent = nil
	}
	key.kind = arr[2]
	return nil
}

func (key CKey) stringSep(separator string) string {
	var p string
	if key.parent == nil {
		p = ""
	} else {
		p = key.parent.stringSep(".")
	}
	return fmt.Sprintf("%v%v%v%v%v", key.id, separator, p, separator, key.kind)
}

func (db inMemoryDb) Get(item interface{}, keyAsString string) (interface{}, error) {
	p := reflect.ValueOf(item)
	if !isValidEntityType(p) {
		return nil, errors.New("Invalid Entity Type: " + p.Kind().String())
	}
	if key, err := db.DecodeKey(keyAsString); err != nil {
		return nil, err
	} else {
		ckey := key.(CKey)
		if kind, ok := db.data[ckey.kind]; !ok {
			return nil, errors.New(fmt.Sprintf("Kind '%v' not found", ckey.kind))
		} else {
			if v, ok := kind.items[ckey.id]; !ok {
				return nil, errors.New(fmt.Sprintf("Id '%v' not found", ckey.id))
			} else {
				reflect.Indirect(p).Set(reflect.Indirect(reflect.ValueOf(v)))
				return v, nil
			}
		}
	}
}

func (db inMemoryDb) GetAll(kind string, ancestor string, items interface{}, filters M, orderKeys []string) (Keys, interface{}, error) {
	return db.GetAllWithLimit(kind, ancestor, items, filters, orderKeys, 0)
}

func (db inMemoryDb) GetAllWithLimit(kind string, ancestor string, items interface{}, filters M, orderKeys []string, limit int) (Keys, interface{}, error) {
	if _, ok := db.data[kind]; !ok {
		return nil, nil, errors.New(fmt.Sprintf("Kind '%v' not found", kind))
	} else {
		keys := Keys{}
		var itemsValue, resultItems reflect.Value
		if items == nil {
			for _, item := range db.data[kind].items {
				t := reflect.ValueOf(item).Type()
				resultItems = reflect.MakeSlice(reflect.SliceOf(t), 0, 0)
				itemsValue = resultItems
				break
			}
		} else {
			itemsValue = reflect.ValueOf(items)
			if itemsValue.Kind() != reflect.Ptr {
				return nil, nil, errors.New("Invalid entity type: " + itemsValue.Kind().String())
			}
			itemsValue = reflect.Indirect(itemsValue)
			resultItems = reflect.MakeSlice(itemsValue.Type(), 0, 0)
		}
		for _, item := range db.data[kind].items {
			mustAppend := true
			if len(ancestor) > 0 {
				parent := item.(Identifier).GetKey().Parent()
				if parent == nil || parent.Encode() != ancestor {
					mustAppend = false
				}
			}
			if mustAppend {
				iv := reflect.ValueOf(item)
				if itemsValue.Type().Elem().Kind() != reflect.Ptr {
					iv = reflect.Indirect(iv)
				}
				resultItems = reflect.Append(resultItems, iv)
				keys = keys.Append(item.(Identifier).GetKey())
			}
		}
		if filters != nil {
			if filteredKeys, filteredItems, err := filter(db, keys, resultItems.Interface(), filters); err != nil {
				return nil, nil, err
			} else {
				keys = filteredKeys
				resultItems = reflect.ValueOf(filteredItems)
			}
		}
		if orderKeys != nil {
			sort.Sort(byFields{keys, resultItems, orderKeys})
		}
		if limit > 0 {
			limitedKeys := Keys{}
			limitedItems := reflect.MakeSlice(itemsValue.Type(), 0, 0)
			for i := 0; i < xmath.Min(limit, resultItems.Len()); i++ {
				limitedItems = reflect.Append(limitedItems, resultItems.Index(i))
				limitedKeys = limitedKeys.Append(keys.KeyAt(i))
			}
			keys = limitedKeys
			resultItems = limitedItems
		}
		if items != nil {
			itemsValue.Set(resultItems)
		}
		return keys, items, nil
	}
}

func (db inMemoryDb) GetAllFromCache(kind string, ancestor string, items interface{}, filters M, orderKeys []string, c cache.Cache, cacheKey string) (Keys, interface{}, error) {
	arr := []interface{}{}
	err := c.Get(cacheKey, &arr)
	if err == nil && len(arr) == 0 {
		if keys, _, err := db.GetAll(kind, ancestor, items, filters, orderKeys); err != nil {
			return nil, nil, err
		} else {
			c.Set(cacheKey, []interface{}{keys, items})
		}
	} else if err != nil {
		return nil, nil, err
	}
	if err := c.Get(cacheKey, &arr); err != nil {
		return nil, nil, err
	}
	reflect.Indirect(reflect.ValueOf(items)).Set(reflect.Indirect(reflect.ValueOf(arr[1])))
	return arr[0].(Keys), arr[1], nil
}

func (db inMemoryDb) Save(item interface{}, kind string, ancestor string, param map[string]string) (key Key, err error) {
	p := reflect.ValueOf(item)
	if !isValidEntityType(p) {
		return nil, errors.New("Invalid Entity Type: " + p.Kind().String())
	}
	if _, ok := db.data[kind]; !ok {
		db.data[kind] = &kindItems{items: map[int]interface{}{}}
	}
	items := db.data[kind]
	if item.(Identifier).GetKey().IsZero() {
		items.lastId++
		items.items[items.lastId] = item
		var parent *CKey
		if len(ancestor) > 0 {
			if k, err := db.DecodeKey(ancestor); err != nil {
				return nil, err
			} else {
				ckey := k.(CKey)
				parent = &ckey
			}
		}
		key = CKey{items.lastId, parent, kind}
		item.(Identifier).SetKey(key)
	} else {
		key = item.(Identifier).GetKey()
		items.items[key.(CKey).id] = item
	}
	return key, nil
}

func (db inMemoryDb) Delete(key Key) error {
	ckey := key.(CKey)
	if kind, ok := db.data[ckey.kind]; !ok {
		return errors.New(fmt.Sprintf("Kind '%v' not found", ckey.kind))
	} else {
		if _, ok := kind.items[ckey.id]; !ok {
			return errors.New(fmt.Sprintf("Id '%v' not found", ckey.id))
		}
		delete(kind.items, ckey.id)
	}
	return nil
}

func (db inMemoryDb) Execute(f func(db.Db) error) error {
	return f(db)
}

func (db inMemoryDb) DecodeKey(s string) (Key, error) {
	result := CKey{}
	if err := result.unmarshalJSONSep([]byte(s), "-"); err != nil {
		return nil, err
	}
	return result, nil
}

func (db inMemoryDb) NewKey() Key {
	return CKey{}
}

func (db inMemoryDb) NewStringKey(kind, key string) Key {
	return nil
}

func isValidEntityType(p reflect.Value) bool {
	return p.Kind() == reflect.Ptr && !p.IsNil() && p.Elem().Kind() == reflect.Struct
}
