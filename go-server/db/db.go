package db

import (
	"errors"
	"fmt"
	"github.com/mcesarhm/geek-accounting/go-server/cache"
	"github.com/mcesarhm/geek-accounting/go-server/util"
	"reflect"
	//"log"

	"strings"
	"time"
)

type Db interface {
	Get(item interface{}, keyAsString string) (interface{}, error)
	GetAll(kind string, ancestor string, items interface{}, filters M, orderKeys []string) (Keys, interface{}, error)
	GetAllWithLimit(kind string, ancestor string, items interface{}, filters M, orderKeys []string, limit int) (Keys, interface{}, error)
	GetAllFromCache(kind string, ancestor string, items interface{}, filters M, orderKeys []string, c cache.Cache, cacheKey string) (Keys, interface{}, error)
	Save(item interface{}, kind string, ancestor string, param map[string]string) (key Key, err error)
	Delete(Key) error
	Execute(func() error) error
	DecodeKey(string) (Key, error)
	NewKey() Key
	NewStringKey(kind, key string) Key
}

type Key interface {
	String() string
	Encode() string
	Parent() Key
	IsZero() bool
	MarshalJSON() ([]byte, error)
	UnmarshalJSON([]byte) error
}

type Keys []CKey

func (keys Keys) Len() int {
	return len(keys)
}

func (keys Keys) KeyAt(i int) Key {
	return keys[i]
}

func (keys Keys) Append(key Key) Keys {
	return Keys(append(keys, key.(CKey)))
}

func (keys Keys) Swap(i, j int) {
	keys[i], keys[j] = keys[j], keys[i]
}

type M map[string]interface{}

type Identifiable struct {
	Key CKey `datastore:"-" json:"_id"`
}

type Identifier interface {
	GetKey() Key
	SetKey(Key)
}

func (identifiable *Identifiable) GetKey() Key {
	return identifiable.Key
}

func (identifiable *Identifiable) SetKey(key Key) {
	identifiable.Key = key.(CKey)
	return
}

type ValidationMessager interface {
	ValidationMessage(Db, map[string]string) string
}

func keysAsStrings(keys Keys) []string {
	result := []string{}
	for i := 0; i < keys.Len(); i++ {
		result = append(result, keys.KeyAt(i).Encode())
	}
	return result
}

func stringsAsKeys(db Db, strings []string) (Keys, error) {
	result := Keys{}
	for _, s := range strings {
		if key, err := db.DecodeKey(s); err != nil {
			return nil, err
		} else {
			result = result.Append(key)
		}
	}
	return result, nil
}

type comparable interface {
	Less(i comparable) bool
}

type intComparable int

func (a intComparable) Less(b comparable) bool {
	return a < b.(intComparable)
}

type stringComparable string

func (a stringComparable) Less(b comparable) bool {
	return a < b.(stringComparable)
}

type timeComparable time.Time

func (a timeComparable) Less(b comparable) bool {
	return time.Time(a).Before(time.Time(b.(timeComparable)))
}

type byFields struct {
	keys   Keys
	values reflect.Value
	fields []string
}

func (a byFields) Len() int { return a.values.Len() }

func (a byFields) Swap(i, j int) {
	a.keys.Swap(i, j)
	swap(a.values, i, j)
}

func (a byFields) Less(i, j int) bool {
	for _, f := range a.fields {
		less := "<"
		greater := ">"
		if f[0:1] == "-" {
			f = f[1:len(f)]
			less = ">"
			greater = "<"
		}
		vi := reflect.Indirect(a.values.Index(i)).FieldByName(f).Interface()
		vj := reflect.Indirect(a.values.Index(j)).FieldByName(f).Interface()
		if ok, _ := compare(vi, vj, greater); ok {
			return false
		}
		if ok, _ := compare(vi, vj, less); ok {
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

func filter(d Db, keys Keys, items interface{}, filters map[string]interface{}) (Keys, interface{}, error) {
	resultKeys := Keys{}
	iv := reflect.ValueOf(items)
	if iv.IsNil() {
		return nil, nil, errors.New("Invalid entity type")
	}
	iv = reflect.Indirect(iv)
	resultItems := reflect.MakeSlice(iv.Type(), 0, 0)
	ii := reflect.ValueOf(items)
	for i := 0; i < ii.Len(); i++ {
		item := ii.Index(i)
		if ii.Type().Elem().Kind() != reflect.Ptr {
			item = reflect.Indirect(item)
		}
		if ok, err := Matches(item.Interface(), filters); err != nil {
			return nil, nil, err
		} else if ok {
			resultKeys = resultKeys.Append(keys.KeyAt(i))
			resultItems = reflect.Append(resultItems, item)
		}
	}
	return resultKeys, resultItems.Interface(), nil
}

func Matches(value interface{}, filters map[string]interface{}) (bool, error) {
	item := reflect.Indirect(reflect.ValueOf(value))
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
	var c1, c2 comparable
	if intValue, ok := v1.(int); ok {
		c1 = intComparable(intValue)
		c2 = intComparable(v2.(int))
	} else if stringValue, ok := v1.(string); ok {
		c1 = stringComparable(stringValue)
		c2 = stringComparable(v2.(string))
	} else if timeValue, ok := v1.(time.Time); ok {
		c1 = timeComparable(timeValue)
		c2 = timeComparable(v2.(time.Time))
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
