/*
	TODO

	- deixar de identificar se uma conta tem parent pela presença do ponto e passar a usar um campo parent
	- trocar os tipos dos campos Key para datastore.Key e retirá-los do bd
	- gravar a chave do usuário na conta e na transação
*/

package domain

import (
	//"log"
	"fmt"
	"reflect"
	"strings"
	"appengine"
	"appengine/datastore"
)

type ChartOfAccounts struct {
	Key string `json:"_id"`
	Name string `json:"name"`
	RetainedEarningsAccount *datastore.Key `json:"retainedEarningsAccount"`
	User *datastore.Key `json:"user"`
}

func (coa *ChartOfAccounts) ValidationMessage(_ appengine.Context, _ map[string]string) string {
	if len(strings.TrimSpace(coa.Name)) == 0 {
		return "The name must be informed"
	}
	return ""
}

type Account struct {
	Key string `json:"_id"`
	Number string `json:"number"`
	Name string `json:"name"`
	Tags []string `json:"tags"`
}

var inheritedProperties = map[string]string{
	"balanceSheet": "financial statement",
	"incomeStatement": "financial statement",
	"operating": "income statement attribute",
	"deduction": "income statement attribute",
	"salesTax": "income statement attribute",
	"cost": "income statement attribute",
	"nonOperatingTax": "income statement attribute",
	"incomeTax": "income statement attribute",
	"dividends": "income statement attribute"}

func (account *Account) ValidationMessage(c appengine.Context, param map[string]string) string {
	if len(strings.TrimSpace(account.Number)) == 0 {
		return "The number must be informed"
	}
	if len(strings.TrimSpace(account.Name)) == 0 {
		return "The name must be informed"
	}
	if !contains(account.Tags, "balanceSheet") && !contains(account.Tags, "incomeStatement") {
		return "The financial statement must be informed"
	}
	if contains(account.Tags, "balanceSheet") && contains(account.Tags, "incomeStatement") {
		return "The statement must be either balance sheet or income statement"
	}
	if !contains(account.Tags, "debitBalance") && !contains(account.Tags, "creditBalance") {
		return "The normal balance must be informed"
	}
	if contains(account.Tags, "debitBalance") && contains(account.Tags, "creditBalance") {
		return "The normal balance must be either debit or credit"
	}
	count := 0
	for _, p := range account.Tags {
		if inheritedProperties[p] == "income statement attribute" {
			count++
		}
	}
	if count > 1 {
		return "Only one income statement attribute is allowed"
	}
	if strings.Contains(account.Number, ".") {
		parentNumber := account.Number[:strings.LastIndex(account.Number, ".")]
		coaKey, err := datastore.DecodeKey(param["coa"])
		if err != nil {
			return err.Error()
		}
		q := datastore.NewQuery("Account").Ancestor(coaKey).Filter("Number = ", parentNumber)
		var parent []Account
		_, err = q.GetAll(c, &parent)
		if err != nil {
			return err.Error()
		}
		if len(parent) == 0 {
			return "Parent not found"
		}
		for key, value := range inheritedProperties {
			if contains(parent[0].Tags, key) && !contains(account.Tags, key) {
				return "The " + value + " must be same as the parent"
			}
		}
	}
	return ""
}

func AllChartsOfAccounts(c appengine.Context, _ map[string]string, _ *datastore.Key) (interface{}, error) {
	return getAll(c, &[]ChartOfAccounts{}, "ChartOfAccounts", "")
}

func SaveChartOfAccounts(c appengine.Context, m map[string]interface{}, param map[string]string, userKey *datastore.Key) (interface{}, error) {
	coa := &ChartOfAccounts{Name: m["name"].(string), User: userKey}
	_, err := save(c, coa, "ChartOfAccounts", "", param)
	return coa, err
}

func AllAccounts(c appengine.Context, param map[string]string, _ *datastore.Key) (interface{}, error) {
	return getAll(c, &[]Account{}, "Account", param["coa"])
}

func SaveAccount(c appengine.Context, m map[string]interface{}, param map[string]string, _ *datastore.Key) (item interface{}, err error) {
	account := &Account{Number: m["number"].(string), Name: m["name"].(string), Tags: []string{}}
	for k, _ := range m {
		if k != "name" && k != "number" {
			account.Tags = append(account.Tags, k)
		}
	}
	if !contains(account.Tags, "analytic") {
		account.Tags = append(account.Tags, "analytic")
	}

	retainedEarningsAccount := contains(account.Tags, "retainedEarnings")
	i := indexOf(account.Tags, "retainedEarnings")
	if i != -1 {
		account.Tags = append(account.Tags[:i], account.Tags[i+1:]...)
	}

	var accountKey *datastore.Key
	accountKey, err = save(c, account, "Account", param["coa"], param)
	if err != nil {
		return
	}

	var coaKey *datastore.Key
	coaKey, err = datastore.DecodeKey(param["coa"])
	if err != nil {
		return
	}

	if retainedEarningsAccount {
		coa := new(ChartOfAccounts)
		if err = datastore.Get(c, coaKey, coa); err != nil {
			return
		}
		coa.RetainedEarningsAccount = accountKey
		if _, err = datastore.Put(c, coaKey, coa); err != nil {
			return
		}
	}

	if strings.Contains(account.Number, ".") {
		parentNumber := account.Number[:strings.LastIndex(account.Number, ".")]
		q := datastore.NewQuery("Account").Ancestor(coaKey).Filter("Number = ", parentNumber)
		var parent []Account
		var keys []*datastore.Key
		keys, err = q.GetAll(c, &parent)
		if err != nil {
			return
		}
		i := indexOf(parent[0].Tags, "analytic")
		if i != -1 {
			parent[0].Tags = append(parent[0].Tags[:i], parent[0].Tags[i+1:]...)
		}
		parent[0].Tags = append(parent[0].Tags, "synthetic")
		if _, err = datastore.Put(c, keys[0], &parent[0]); err != nil {
			return
		}
	}

	item = account
	return
}

func getAll(c appengine.Context, items interface{}, kind string, ancestor string) (interface{}, error) {
	q := datastore.NewQuery(kind)
	if len(ancestor) > 0 {
		ancestorKey, err := datastore.DecodeKey(ancestor)
		if err != nil {
			return nil, err
		}
		q = q.Ancestor(ancestorKey)
	}
	keys, err := q.GetAll(c, items)
	v := reflect.ValueOf(items).Elem()
	for i := 0; i < v.Len(); i++ {
		v.Index(i).FieldByName("Key").SetString(keys[i].Encode())
	}
	return items, err
}

func save(c appengine.Context, item interface{}, kind string, ancestor string, param map[string]string) (key *datastore.Key, err error) {
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

	if len(v.FieldByName("Key").String()) > 0 {
		key, err = datastore.DecodeKey(v.FieldByName("Key").String())
		if err != nil {
			return
		}
	} else {
		key = datastore.NewIncompleteKey(c, kind, ancestorKey)
	}

	key, err = datastore.Put(c, key, item)
	if err != nil {
		return
	}

	if key != nil {
		v.FieldByName("Key").SetString(key.Encode())
	}

	return
}

func contains(s []string, e string) bool {
    return indexOf(s, e) != -1
}

func indexOf(s []string, e string) int {
    for i, a := range s { if a == e { return i } }
    return -1
}

type ValidationMessager interface {
	ValidationMessage(appengine.Context, map[string]string) string
}
