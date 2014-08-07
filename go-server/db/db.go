package db

import (
	"appengine"
	"appengine/datastore"
	"appengine/memcache"
	"fmt"
	"github.com/mcesarhm/geek-accounting/go-server/util"
	"reflect"
	"sort"
	"strings"
	"time"
)

func GetAll(c appengine.Context, items interface{}, kind string, ancestor string, orderKeys []string) (interface{}, error) {
	q := datastore.NewQuery(kind)
	if len(ancestor) > 0 {
		ancestorKey, err := datastore.DecodeKey(ancestor)
		if err != nil {
			return nil, err
		}
		q = q.Ancestor(ancestorKey)
	}
	for _, o := range orderKeys {
		q = q.Order(o)
	}
	keys, err := q.GetAll(c, items)
	v := reflect.ValueOf(items).Elem()
	for i := 0; i < v.Len(); i++ {
		v.Index(i).FieldByName("Key").Set(reflect.ValueOf(keys[i]))
	}
	return items, err
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
	v := reflect.ValueOf(item).Elem()
	v.FieldByName("Key").Set(reflect.ValueOf(key))
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

	v := reflect.ValueOf(item).Elem()

	if !v.FieldByName("Key").IsNil() {
		key = v.FieldByName("Key").Interface().(*datastore.Key)
	} else {
		key = datastore.NewIncompleteKey(c, kind, ancestorKey)
	}

	key, err = datastore.Put(c, key, item)
	if err != nil {
		return
	}

	if key != nil {
		v.FieldByName("Key").Set(reflect.ValueOf(key))
	}

	return
}

type ValidationMessager interface {
	ValidationMessage(appengine.Context, map[string]string) string
}

func GetWithCache(c appengine.Context, cacheKey, kind string, ancestor *datastore.Key, order []string, filters map[string]interface{}, items interface{}, filteredItems interface{}) ([]*datastore.Key, interface{}, error) {
	var arr []interface{}
	var keys []*datastore.Key
	_, err := memcache.Gob.Get(c, cacheKey, &arr)
	if err == memcache.ErrCacheMiss {
		q := datastore.NewQuery(kind)
		if ancestor != nil {
			q = q.Ancestor(ancestor)
		}
		/*
			if order != nil {
				for _, fieldName := range order {
					q = q.Order(fieldName)
				}
			}
		*/
		if keys, err = q.GetAll(c, items); err != nil {
			return nil, nil, err
		}
		items = reflect.Indirect(reflect.ValueOf(items)).Interface()
		keysAsString := []string{}
		for _, k := range keys {
			keysAsString = append(keysAsString, k.Encode())
		}
		item := &memcache.Item{
			Key:    cacheKey,
			Object: []interface{}{keysAsString, items},
		}
		if err = memcache.Gob.Set(c, item); err != nil {
			return nil, nil, err
		}
	} else if err != nil {
		return nil, nil, err
	} else {
		keysAsString := arr[0].([]string)
		items = arr[1]
		keys = []*datastore.Key{}
		for _, k := range keysAsString {
			if key, err := datastore.DecodeKey(k); err != nil {
				return nil, nil, err
			} else {
				keys = append(keys, key)
			}
		}
	}
	if filters != nil {
		if keys, items, err = filter(keys, items, filters, order, filteredItems); err != nil {
			return nil, nil, err
		}
	}
	return keys, items, nil
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
	value  reflect.Value
	fields []string
}

func (a ByFields) Len() int { return a.value.Len() }

func (a ByFields) Swap(i, j int) {
	vi := a.value.Index(i)
	vi.Set(a.value.Index(j))
	a.value.Index(j).Set(vi)
}

func (a ByFields) Less(i, j int) bool {
	for _, f := range a.fields {
		vi := reflect.Indirect(a.value.Index(i)).FieldByName(f)
		vj := reflect.Indirect(a.value.Index(j)).FieldByName(f).Interface()
		if ok, _ := matchesWithOperator(vi, vj, "<"); ok {
			return true
		}
	}
	return false
}

func filter(keys []*datastore.Key, items interface{}, filters map[string]interface{}, order []string, dest interface{}) ([]*datastore.Key, interface{}, error) {
	resultKeys := []*datastore.Key{}
	resultItems := reflect.ValueOf(dest)
	ii := reflect.ValueOf(items)
	for i := 0; i < ii.Len(); i++ {
		itemPtr := ii.Index(i)
		item := itemPtr.Elem()
		matches := true
		for k, v := range filters {
			keyArray := strings.Split(k, " ")
			fn := keyArray[0]
			operator := keyArray[1]
			f := item.FieldByName(fn)
			if f.Kind() == reflect.Slice {
				if !strings.HasSuffix(k, " =") {
					return nil, nil, fmt.Errorf("Operators other than equal are not allowed")
				}
				found := false
				for j := 0; j < f.Len(); j++ {
					if f.Index(j).Interface() == v {
						found = true
						break
					}
				}
				if !found {
					matches = false
					break
				}
			} else if operator == "=" {
				if f.Interface() != v {
					matches = false
					break
				}
			} else if util.Contains([]string{"<", ">", "<=", ">="}, operator) {
				ok, err := matchesWithOperator(f, v, operator)
				if err != nil {
					return nil, nil, err
				}
				if !ok {
					matches = false
					break
				}
			} else {
				return nil, nil, fmt.Errorf("Operator not allowed: " + operator)
			}
		}
		if matches {
			resultKeys = append(resultKeys, keys[i])
			resultItems = reflect.Append(resultItems, itemPtr)
		}
	}
	if order != nil {
		sort.Sort(ByFields{resultItems, order})
	}
	return resultKeys, resultItems.Interface(), nil
}

func matchesWithOperator(f reflect.Value, v interface{}, operator string) (bool, error) {
	var c1, c2 Comparable
	if intValue, ok := f.Interface().(int); ok {
		c1 = IntComparable(intValue)
		c2 = IntComparable(v.(int))
	} else if stringValue, ok := f.Interface().(string); ok {
		c1 = StringComparable(stringValue)
		c2 = StringComparable(v.(string))
	} else if timeValue, ok := f.Interface().(time.Time); ok {
		c1 = TimeComparable(timeValue)
		c2 = TimeComparable(v.(time.Time))
	} else {
		return false, fmt.Errorf("Type not allowed: " + f.Type().String())
	}
	if (operator == "<" || operator == "<=") && c2.Less(c1) {
		return false, nil
	}
	if (operator == ">" || operator == ">=") && c1.Less(c2) {
		return false, nil
	}
	if (operator == "<" || operator == ">") && c1 == c2 {
		return false, nil
	}
	return true, nil
}
