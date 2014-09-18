package db

import (
	"appengine"
	"appengine/datastore"
	"appengine/memcache"
	"fmt"
	"github.com/mcesarhm/geek-accounting/go-server/util"
	//"log"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Identifiable struct {
	Key *datastore.Key `datastore:"-" json:"_id"`
}

type Identifier interface {
	GetKey() *datastore.Key
	SetKey(*datastore.Key)
}

func (identifiable *Identifiable) GetKey() *datastore.Key {
	return identifiable.Key
}

func (identifiable *Identifiable) SetKey(key *datastore.Key) {
	identifiable.Key = key
	return
}

func GetAll(c appengine.Context, kind string, ancestor string, items interface{}, filters map[string]interface{}, orderKeys []string) ([]*datastore.Key, interface{}, error) {
	q := datastore.NewQuery(kind)
	if len(ancestor) > 0 {
		ancestorKey, err := datastore.DecodeKey(ancestor)
		if err != nil {
			return nil, nil, err
		}
		q = q.Ancestor(ancestorKey)
	}
	if filters != nil {
		for _, o := range orderKeys {
			q = q.Order(o)
		}
	}
	if orderKeys != nil {
		for k, v := range filters {
			q = q.Filter(k, v)
		}
	}
	if items == nil {
		q = q.KeysOnly()
	}
	keys, err := q.GetAll(c, items)
	if items != nil {
		v := reflect.ValueOf(items).Elem()
		for i := 0; i < v.Len(); i++ {
			ptr := v.Index(i)
			if ptr.Kind() != reflect.Ptr {
				ptr = ptr.Addr()
			}
			identifier := ptr.Interface().(Identifier)
			identifier.SetKey(keys[i])
		}
	}
	return keys, items, err
}

func Get(c appengine.Context, item interface{}, keyAsString string) (result interface{}, err error) {
	key, err := datastore.DecodeKey(keyAsString)
	if err != nil {
		return
	}
	err = datastore.Get(c, key, item)
	if err != nil {
		return
	}
	identifier := item.(Identifier)
	identifier.SetKey(key)
	result = item
	return
}

func Save(c appengine.Context, item interface{}, kind string, ancestor string, param map[string]string) (key *datastore.Key, err error) {
	vm := item.(ValidationMessager).ValidationMessage(c, param)
	if len(vm) > 0 {
		return nil, fmt.Errorf(vm)
	}

	var ancestorKey *datastore.Key
	if len(ancestor) > 0 {
		ancestorKey, err = datastore.DecodeKey(ancestor)
		if err != nil {
			return
		}
	}

	identifier := item.(Identifier)
	if identifier.GetKey() != nil {
		key = identifier.GetKey()
	} else {
		key = datastore.NewIncompleteKey(c, kind, ancestorKey)
	}

	key, err = datastore.Put(c, key, item)
	if err != nil {
		return
	}

	if key != nil {
		identifier.SetKey(key)
	}

	return
}

type ValidationMessager interface {
	ValidationMessage(appengine.Context, map[string]string) string
}

func GetAllFromCache(c appengine.Context, kind string, ancestor *datastore.Key, items interface{}, filteredItems interface{}, filters map[string]interface{}, order []string, cacheKey string) ([]*datastore.Key, interface{}, error) {
	var arr []interface{}
	var keys []*datastore.Key
	_, err := memcache.Gob.Get(c, cacheKey, &arr)
	if err == memcache.ErrCacheMiss {
		q := datastore.NewQuery(kind)
		if ancestor != nil {
			q = q.Ancestor(ancestor)
		}
		if keys, err = q.GetAll(c, items); err != nil {
			return nil, nil, err
		}
		itemsValue := reflect.Indirect(reflect.ValueOf(items))
		items = itemsValue.Interface()
		nextKey := cacheKey
		nextChunk := 0
		i := 0
		for {
			chunkSize := util.Min(MaxItemsPerMemcacheEntry, len(keys)-i)
			keysChunk := make([]*datastore.Key, chunkSize)
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
			if err = memcache.Gob.Set(c, memcacheItem); err != nil {
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
		chunkKeys, err := stringsAsKeys(arr[0].([]string))
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
		if _, err := memcache.Gob.Get(c, arr[2].(string), &arr); err != nil {
			return nil, nil, err
		}
	}
	items = filteredItems
	if order != nil {
		sort.Sort(ByFields{keys, reflect.ValueOf(items), order})
	}

	return keys, items, nil
}

func keysAsStrings(keys []*datastore.Key) []string {
	result := []string{}
	for _, k := range keys {
		result = append(result, k.Encode())
	}
	return result
}

func stringsAsKeys(strings []string) ([]*datastore.Key, error) {
	result := []*datastore.Key{}
	for _, s := range strings {
		if key, err := datastore.DecodeKey(s); err != nil {
			return nil, err
		} else {
			result = append(result, key)
		}
	}
	return result, nil
}

type Comparable interface {
	Less(i Comparable) bool
}

type IntComparable int

func (a IntComparable) Less(b Comparable) bool {
	return a < b.(IntComparable)
}

type StringComparable string

func (a StringComparable) Less(b Comparable) bool {
	return a < b.(StringComparable)
}

type TimeComparable time.Time

func (a TimeComparable) Less(b Comparable) bool {
	return time.Time(a).Before(time.Time(b.(TimeComparable)))
}

type ByFields struct {
	keys   []*datastore.Key
	values reflect.Value
	fields []string
}

func (a ByFields) Len() int { return a.values.Len() }

func (a ByFields) Swap(i, j int) {
	a.keys[i], a.keys[j] = a.keys[j], a.keys[i]
	swap(a.values, i, j)
}

func (a ByFields) Less(i, j int) bool {
	for _, f := range a.fields {
		vi := a.values.Index(i).Elem().FieldByName(f).Interface()
		vj := a.values.Index(j).Elem().FieldByName(f).Interface()
		if ok, _ := compare(vi, vj, ">"); ok {
			return false
		}
		if ok, _ := compare(vi, vj, "<"); ok {
			return true
		}
	}
	return false
}

func swap(arr reflect.Value, i, j int) {
	t := reflect.ValueOf(arr.Index(i).Interface())
	arr.Index(i).Set(arr.Index(j))
	arr.Index(j).Set(t)
}

func filter(keys []*datastore.Key, items interface{}, filters map[string]interface{}, dest interface{}) ([]*datastore.Key, interface{}, error) {
	resultKeys := []*datastore.Key{}
	resultItems := reflect.ValueOf(dest)
	ii := reflect.ValueOf(items)
	for i := 0; i < ii.Len(); i++ {
		item := ii.Index(i)
		if ok, err := Matches(item.Elem().Interface(), filters); err != nil {
			return nil, nil, err
		} else if ok {
			resultKeys = append(resultKeys, keys[i])
			resultItems = reflect.Append(resultItems, item)
		}
	}
	return resultKeys, resultItems.Interface(), nil
}

func Matches(value interface{}, filters map[string]interface{}) (bool, error) {
	item := reflect.ValueOf(value).Elem()
	for k, v := range filters {
		keyArray := strings.Split(k, " ")
		fn := keyArray[0]
		operator := keyArray[1]
		f := item.FieldByName(fn)
		if f.Kind() == reflect.Slice {
			if !strings.HasSuffix(k, " =") {
				return false, fmt.Errorf("Operators other than equal are not allowed")
			}
			found := false
			for j := 0; j < f.Len(); j++ {
				if f.Index(j).Interface() == v {
					found = true
					break
				}
			}
			if !found {
				return false, nil
			}
		} else if operator == "=" {
			if f.Interface() != v {
				return false, nil
			}
		} else if util.Contains([]string{"<", ">", "<=", ">="}, operator) {
			ok, err := compare(f.Interface(), v, operator)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		} else {
			return false, fmt.Errorf("Operator not allowed: " + operator)
		}
	}
	return true, nil
}

func compare(v1, v2 interface{}, operator string) (bool, error) {
	var c1, c2 Comparable
	if intValue, ok := v1.(int); ok {
		c1 = IntComparable(intValue)
		c2 = IntComparable(v2.(int))
	} else if stringValue, ok := v1.(string); ok {
		c1 = StringComparable(stringValue)
		c2 = StringComparable(v2.(string))
	} else if timeValue, ok := v1.(time.Time); ok {
		c1 = TimeComparable(timeValue)
		c2 = TimeComparable(v2.(time.Time))
	} else {
		return false, fmt.Errorf("Type not allowed: %v", reflect.ValueOf(v1).Type())
	}
	if (operator == "<") && !c1.Less(c2) {
		return false, nil
	}
	if (operator == "<=") && c2.Less(c1) {
		return false, nil
	}
	if (operator == ">") && !c2.Less(c1) {
		return false, nil
	}
	if (operator == ">=") && c1.Less(c2) {
		return false, nil
	}
	return true, nil
}
