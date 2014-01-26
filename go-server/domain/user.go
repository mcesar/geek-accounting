package domain

import (
	"crypto/sha1"
	"fmt"
	//"log"
	"appengine"
	"appengine/datastore"
)

type User struct {
	User string `json:"user"`
	Name string `json:"name"`
	Password string `json:-`
}

func InitUserManagement(c appengine.Context) (err error) {
	var user *User
	err, user, _ = userByLogin(c, "admin")
	if err != nil {
		return
	}
	if user == nil {
		key := datastore.NewIncompleteKey(c, "User", nil)
		_, err = datastore.Put(c, key, &User{User: "admin", Password: hash("admin")})
		if err != nil {
			return
		}
	}
	return
}

func Login(c appengine.Context, login, password string) (error, bool, *datastore.Key) {
	err, user, key := userByLogin(c, login)
	if err != nil {
		return err, false, nil
	}
	if user == nil || user.Password != hash(password) {
		return nil, false, nil
	}
	return nil, true, key
}

func hash(s string) string {
	return fmt.Sprintf("%x", sha1.New().Sum([]byte(s)))
}

func userByLogin(c appengine.Context, login string) (err error, user *User, key *datastore.Key) {
	q := datastore.NewQuery("User").Filter("User = ", login)
	var users []User
	keys, err := q.GetAll(c, &users)
	if err != nil {
		return
	}
	if len(users) == 0 {
		return
	}
	user = &users[0]
	key = keys[0]
	return
}