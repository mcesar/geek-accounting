package db

import (
	"fmt"
	"github.com/mcesarhm/geek-accounting/go-server/util"
	//"log"
	"reflect"
	"strings"
	"time"
)

type Db interface {
	Get(item interface{}, keyAsString string) (interface{}, error)
	GetAll(kind string, ancestor string, items interface{}, filters M, orderKeys []string) (Keys, interface{}, error)
	GetAllWithLimit(kind string, ancestor string, items interface{}, filters M, orderKeys []string, limit int) (Keys, interface{}, error)
	GetAllFromCache(kind string, ancestor Key, items interface{}, filteredItems interface{}, filters M, order []string, cacheKey string) (Keys, interface{}, error)
	Save(item interface{}, kind string, ancestor string, param map[string]string) (key Key, err error)
	Delete(Key) error
	Execute(func() error) error
	DecodeKey(string) (Key, error)
	NewStringKey(kind, key string) Key
}

type M map[string]interface{}

type Identifiable struct {
	Key Key `datastore:"-" json:"_id"`
}

type Identifier interface {
	GetKey() Key
	SetKey(Key)
}

func (identifiable *Identifiable) GetKey() Key {
	return identifiable.Key
}

func (identifiable *Identifiable) SetKey(key Key) {
	identifiable.Key = key
	return
}

type ValidationMessager interface {
	ValidationMessage(Db, map[string]string) string
}

func keysAsStrings(keys Keys) []string {
	result := []string{}
	for _, k := range keys {
		result = append(result, k.Encode())
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
	keys   Keys
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

func filter(keys Keys, items interface{}, filters map[string]interface{}, dest interface{}) (Keys, interface{}, error) {
	resultKeys := Keys{}
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
