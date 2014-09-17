package core

import (
	"crypto/sha1"
	"fmt"
	//"log"
	"appengine"
	"appengine/datastore"
	"github.com/mcesarhm/geek-accounting/go-server/db"
	"strings"
)

type User struct {
	Key      *datastore.Key `datastore:"-" json:"_id"`
	User     string         `json:"user"`
	Name     string         `json:"name"`
	Password string         `json:"-"`
}

func (u *User) ValidationMessage(_ appengine.Context, m map[string]string) string {
	if len(strings.TrimSpace(u.User)) == 0 {
		return "The login must be informed"
	}
	if len(strings.TrimSpace(u.Name)) == 0 {
		return "The name must be informed"
	}
	return ""
}

func InitUserManagement(c appengine.Context) (err error) {
	var user *User
	err, user, _ = userByLogin(c, "admin", true)
	if err != nil {
		return
	}
	if user == nil {
		key := datastore.NewIncompleteKey(c, "User", realm(c))
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

func ChangePassword(c appengine.Context, m map[string]interface{}, _ map[string]string, userKey *datastore.Key) (item interface{}, err error) {
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

func AllUsers(c appengine.Context, _ map[string]string, _ *datastore.Key) (interface{}, error) {
	users, _, err := db.GetAll(c, "User", realm(c).Encode(), &[]User{}, nil, []string{"User"})
	return users, err
}

func GetUser(c appengine.Context, param map[string]string, _ *datastore.Key) (interface{}, error) {
	return db.Get(c, &User{}, param["user"])
}

func SaveUser(c appengine.Context, m map[string]interface{}, param map[string]string, userKey *datastore.Key) (item interface{}, err error) {

	user := &User{
		User: m["user"].(string),
		Name: m["name"].(string),
	}

	if userKeyAsString, ok := param["user"]; ok {
		user.Key, err = datastore.DecodeKey(userKeyAsString)
		if err != nil {
			return
		}
		if password, ok := m["password"]; !ok || len(password.(string)) == 0 {
			var u User
			err = datastore.Get(c, user.Key, &u)
			if err != nil {
				return
			}
			user.Password = u.Password
		} else {
			user.Password = hash(password.(string))
		}
	} else {
		user.Password = hash(m["password"].(string))
	}

	if user.Key, err = db.Save(c, user, "User", realm(c).Encode(), param); err != nil {
		return
	}

	item = user

	return
}

func DeleteUser(c appengine.Context, m map[string]interface{}, param map[string]string, userKey *datastore.Key) (_ interface{}, err error) {

	key, err := datastore.DecodeKey(param["user"])
	if err != nil {
		return
	}

	err = datastore.Delete(c, key)

	return

}

func hash(s string) string {
	return fmt.Sprintf("%x", sha1.New().Sum([]byte(s)))
}

func userByLogin(c appengine.Context, login string, init bool) (err error, user *User, key *datastore.Key) {
	q := datastore.NewQuery("User").Ancestor(realm(c)).Filter("User = ", login)
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

func realm(c appengine.Context) *datastore.Key {
	return datastore.NewKey(c, "User", "default", 0, nil)
}
