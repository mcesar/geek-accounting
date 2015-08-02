package core

import (
	"crypto/sha1"
	"fmt"
	//"log"
	"github.com/mcesarhm/geek-accounting/go-server/context"
	"github.com/mcesarhm/geek-accounting/go-server/db"
	"strings"
)

type UserKey db.CKey

func NewUserKey() UserKey {
	return UserKey(*new(db.CKey))
}

func (key UserKey) MarshalJSON() ([]byte, error) {
	return db.CKey(key).MarshalJSON()
}

func (key UserKey) UnmarshalJSON(b []byte) error {
	return db.CKey(key).UnmarshalJSON(b)
}

func (key UserKey) Encode() string {
	return db.CKey(key).Encode()
}

type User struct {
	db.Identifiable
	User     string `json:"user"`
	Name     string `json:"name"`
	Password string `json:"-"`
}

func (u *User) ValidationMessage(_ db.Db, _ map[string]string) string {
	if len(strings.TrimSpace(u.User)) == 0 {
		return "The login must be informed"
	}
	if len(strings.TrimSpace(u.Name)) == 0 {
		return "The name must be informed"
	}
	return ""
}

func InitUserManagement(c context.Context) (err error) {
	var user *User
	err, user, _ = userByLogin(c, "admin", true)
	if err != nil {
		return
	}
	if user == nil {
		_, err = c.Db.Save(&User{User: "admin", Password: hash("admin"), Name: "admin"},
			"User", realm(c.Db), nil)
		if err != nil {
			return
		}
	}
	return
}

func Login(c context.Context, login, password string) (error, bool, UserKey) {
	err, user, key := userByLogin(c, login, false)
	if err != nil {
		return err, false, UserKey{}
	}
	if user == nil || user.Password != hash(password) {
		return nil, false, UserKey{}
	}
	return nil, true, key
}

func ChangePassword(c context.Context, m map[string]interface{}, _ map[string]string, userKey UserKey) (item interface{}, err error) {
	user := User{}
	_, err = c.Db.Get(&user, userKey.Encode())
	if err != nil {
		return
	}
	if user.Password != hash(m["oldPassword"].(string)) {
		return nil, fmt.Errorf("Wrong old password")
	}
	user.Password = hash(m["newPassword"].(string))
	_, err = c.Db.Save(&user, "User", realm(c.Db), nil)
	return
}

func AllUsers(c context.Context, _ map[string]interface{}, _ map[string]string,
	_ UserKey) (interface{}, error) {
	users, _, err := c.Db.GetAll("User", realm(c.Db), &[]User{}, nil, []string{"User"})
	return users, err
}

func GetUser(c context.Context, _ map[string]interface{}, param map[string]string,
	_ UserKey) (interface{}, error) {
	return c.Db.Get(&User{}, param["user"])
}

func SaveUser(c context.Context, m map[string]interface{}, param map[string]string, userKey UserKey) (item interface{}, err error) {

	user := &User{
		User: m["user"].(string),
		Name: m["name"].(string),
	}

	if userKeyAsString, ok := param["user"]; ok {
		if k, err := c.Db.DecodeKey(userKeyAsString); err != nil {
			return nil, err
		} else {
			user.SetKey(k)
		}
		if password, ok := m["password"]; !ok || len(password.(string)) == 0 {
			var u User
			_, err = c.Db.Get(&u, userKeyAsString)
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

	if k, err := c.Db.Save(user, "User", realm(c.Db), param); err != nil {
		return nil, err
	} else {
		user.SetKey(k)
	}

	item = user

	return
}

func DeleteUser(c context.Context, m map[string]interface{}, param map[string]string, userKey UserKey) (_ interface{}, err error) {
	if key, err := c.Db.DecodeKey(param["user"]); err != nil {
		return nil, err
	} else {
		return nil, c.Db.Delete(key)
	}
}

func hash(s string) string {
	return fmt.Sprintf("%x", sha1.New().Sum([]byte(s)))
}

func userByLogin(c context.Context, login string, init bool) (err error, user *User, key UserKey) {
	var users []User
	keys, _, err := c.Db.GetAll("User", realm(c.Db), &users, map[string]interface{}{"User = ": login}, nil)
	if err != nil {
		return
	}
	if len(users) == 0 {
		if login == "admin" && !init {
			if err = InitUserManagement(c); err != nil {
				return
			}
			keys, _, err = c.Db.GetAll("User", realm(c.Db), &users, map[string]interface{}{"User = ": login}, nil)
			if err != nil {
				return
			}
		}
		return
	}
	user = &users[0]
	key = UserKey(keys.KeyAt(0).(db.CKey))
	return
}

func realm(db db.Db) string {
	return db.NewStringKey("User", "default").Encode()
}
