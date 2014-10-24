// +build inmemory

package db

import (
	"github.com/mcesarhm/geek-accounting/go-server/cache"
	"testing"
)

type S struct {
	Identifiable
	Name string
}

type T struct {
	Identifiable
	Name string
}

func TestGet(t *testing.T) {
	db := NewInMemoryDb()
	if key, err := db.Save(&S{Name: "a"}, "S", "", nil); err != nil {
		t.Fatal(err)
	} else {
		var s S
		if s1, err := db.Get(&s, key.Encode()); err != nil {
			t.Fatal(err)
		} else if s.Name != "a" || s1.(*S).Name != "a" {
			t.Error("a expected got", s.Name, s1.(*S).Name)
		}
	}
}

func TestGetAll(t *testing.T) {
	db := NewInMemoryDb()
	if _, err := save(db, "S", "", &S{Name: "a"}, &S{Name: "b"}); err != nil {
		t.Fatal(err)
	}
	var s []S
	if _, _, err := db.GetAll("S", "", &s, nil, nil); err != nil {
		t.Fatal(err)
	} else {
		if len(s) != 2 {
			t.Error("2 expected got", len(s))
		} else if s[0].Name != "a" {
			t.Error("a expected got", s[0].Name)
		} else if s[1].Name != "b" {
			t.Error("b expected got", s[1].Name)
		}
	}
	if _, _, err := db.GetAll("S", "", &s, M{"Name = ": "a"}, nil); err != nil {
		t.Fatal(err)
	} else {
		if len(s) != 1 {
			t.Error("1 expected got", len(s))
		} else if s[0].Name != "a" {
			t.Error("a expected got", s[0].Name)
		}
	}
	if _, _, err := db.GetAllWithLimit("S", "", &s, nil, nil, 1); err != nil {
		t.Fatal(err)
	} else {
		if len(s) != 1 {
			t.Error("1 expected got", len(s))
		} else if s[0].Name != "a" {
			t.Error("a expected got", s[0].Name)
		}
	}
	var (
		keys Keys
		err  error
	)
	if keys, err = save(db, "T", "", &T{Name: "t"}); err != nil {
		t.Fatal(err)
	}
	if _, err = save(db, "S", keys.KeyAt(0).Encode(), &S{Name: "c"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := db.GetAll("S", keys.KeyAt(0).Encode(), &s, nil, nil); err != nil {
		t.Fatal(err)
	} else {
		if len(s) != 1 {
			t.Error("1 expected got", len(s))
		} else if s[0].Name != "c" {
			t.Error("c expected got", s[0].Name)
		}
	}
	var s_ []*S
	if _, _, err := db.GetAll("S", "", &s_, nil, nil); err != nil {
		t.Fatal(err)
	} else {
		if len(s_) != 3 {
			t.Error("3 expected got", len(s_))
		} else if s_[0].Name != "a" {
			t.Error("a expected got", s_[0].Name)
		} else if s_[1].Name != "b" {
			t.Error("b expected got", s_[1].Name)
		} else if s_[2].Name != "c" {
			t.Error("c expected got", s_[2].Name)
		}
	}
	if _, _, err := db.GetAll("S", "", &s, nil, []string{"Name"}); err != nil {
		t.Fatal(err)
	} else {
		if len(s) != 3 {
			t.Error("3 expected got", len(s))
		} else if s[0].Name != "a" {
			t.Error("a expected got", s[0].Name)
		} else if s[1].Name != "b" {
			t.Error("b expected got", s[1].Name)
		} else if s[2].Name != "c" {
			t.Error("c expected got", s[2].Name)
		}
	}
	if _, _, err := db.GetAll("S", "", &s, nil, []string{"-Name"}); err != nil {
		t.Fatal(err)
	} else {
		if len(s) != 3 {
			t.Error("3 expected got", len(s))
		} else if s[0].Name != "c" {
			t.Error("c expected got", s[0].Name)
		} else if s[1].Name != "b" {
			t.Error("b expected got", s[1].Name)
		} else if s[2].Name != "a" {
			t.Error("a expected got", s[2].Name)
		}
	}
}

func TestGetAllFromCache(t *testing.T) {
	db := NewInMemoryDb()
	c := cache.NewInMemoryCache()
	if _, err := save(db, "S", "", &S{Name: "a"}, &S{Name: "b"}); err != nil {
		t.Fatal(err)
	}
	var s []S
	if _, _, err := db.GetAllFromCache("S", "", &s, nil, nil, c, "k"); err != nil {
		t.Fatal(err)
	} else {
		if len(s) != 2 {
			t.Error("2 expected got", len(s))
		} else if s[0].Name != "a" {
			t.Error("a expected got", s[0].Name)
		} else if s[1].Name != "b" {
			t.Error("b expected got", s[1].Name)
		}
	}
}

func TestDelete(t *testing.T) {
	db := NewInMemoryDb()
	if keys, err := save(db, "S", "", &S{Name: "a"}); err != nil {
		t.Fatal(err)
	} else {
		if err := db.Delete(keys.KeyAt(0)); err != nil {
			t.Fatal(err)
		}
		var s []S
		if _, _, err := db.GetAll("S", "", &s, nil, nil); err != nil {
			t.Fatal(err)
		}
		if len(s) != 0 {
			t.Error("0 expected got", len(s))
		}
	}
}

func save(db Db, kind, ancestor string, items ...interface{}) (Keys, error) {
	keys := Keys{}
	for _, i := range items {
		if key, err := db.Save(i, kind, ancestor, nil); err != nil {
			return nil, err
		} else {
			keys = keys.Append(key)
		}
	}
	return keys, nil
}
