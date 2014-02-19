package db

import (
	"appengine"
	"appengine/datastore"
	"fmt"
	"reflect"
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
