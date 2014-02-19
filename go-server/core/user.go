package core

import (
	"crypto/sha1"
	"fmt"
	//"log"
	"appengine"
	"appengine/datastore"
)

type User struct {
	User     string `json:"user"`
	Name     string `json:"name"`
	Password string `json:-`
}

func InitUserManagement(c appengine.Context) (err error) {
	var user *User
	err, user, _ = userByLogin(c, "admin", true)
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
	err, user, key := userByLogin(c, login, false)
	if err != nil {
		return err, false, nil
	}
	if user == nil || user.Password != hash(password) {
		return nil, false, nil
	}
	return nil, true, key
}

func ChangePassword(c appengine.Context, m map[string]interface{}, param map[string]string, userKey *datastore.Key) (item interface{}, err error) {
	user := User{}
	err = datastore.Get(c, userKey, &user)
	if err != nil {
		return
	}
	if user.Password != hash(m["oldPassword"].(string)) {
		return nil, fmt.Errorf("Wrong old password")
	}
	user.Password = hash(m["newPassword"].(string))
	_, err = datastore.Put(c, userKey, &user)
	return
}

func hash(s string) string {
	return fmt.Sprintf("%x", sha1.New().Sum([]byte(s)))
}

func userByLogin(c appengine.Context, login string, init bool) (err error, user *User, key *datastore.Key) {
	q := datastore.NewQuery("User").Filter("User = ", login)
	var users []User
	keys, err := q.GetAll(c, &users)
	if err != nil {
		return
	}
	if len(users) == 0 {
		if login == "admin" && !init {
			if err = InitUserManagement(c); err != nil {
				return
			}
			keys, err = q.GetAll(c, &users)
			if err != nil {
				return
			}
		}
		return
	}
	user = &users[0]
	key = keys[0]
	return
}
