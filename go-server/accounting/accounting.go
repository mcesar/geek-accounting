package accounting

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"strconv"

	"github.com/mcesarhm/geek-accounting/go-server/context"
	"github.com/mcesarhm/geek-accounting/go-server/core"
	"github.com/mcesarhm/geek-accounting/go-server/db"
	"github.com/mcesarhm/geek-accounting/go-server/extensions/collections"
	xmath "github.com/mcesarhm/geek-accounting/go-server/extensions/math"

	"sort"
	"strings"
	"time"

	"mcesar.io/deb"
)

type ChartOfAccounts struct {
	db.Identifiable
	Name                    string       `json:"name"`
	RetainedEarningsAccount db.CKey      `json:"retainedEarningsAccount"`
	Space                   db.CKey      `json:"space"`
	User                    core.UserKey `json:"user"`
	AsOf                    time.Time    `json:"timestamp"`
}

func (coa *ChartOfAccounts) ValidationMessage(_ db.Db, _ map[string]string) string {
	if len(strings.TrimSpace(coa.Name)) == 0 {
		return "The name must be informed"
	}
	return ""
}

type Account struct {
	db.Identifiable
	Number  string       `json:"number"`
	Name    string       `json:"name"`
	Tags    []string     `json:"tags"`
	Parent  db.CKey      `json:"parent"`
	User    core.UserKey `json:"user"`
	AsOf    time.Time    `json:"timestamp"`
	Created time.Time    `json:"-"`
	Removed bool         `json:"-"`
}

var inheritedProperties = map[string]string{
	"balanceSheet":    "financial statement",
	"incomeStatement": "financial statement",
	"operating":       "income statement attribute",
	"deduction":       "income statement attribute",
	"salesTax":        "income statement attribute",
	"cost":            "income statement attribute",
	"nonOperatingTax": "income statement attribute",
	"incomeTax":       "income statement attribute",
	"dividends":       "income statement attribute",
}

var nonInheritedProperties = map[string]string{
	"debitBalance":  "",
	"creditBalance": "",
	"analytic":      "",
	"synthetic":     "",
}

func (account *Account) ValidationMessage(db db.Db, param map[string]string) string {
	if len(strings.TrimSpace(account.Number)) == 0 {
		return "The number must be informed"
	}
	if len(strings.TrimSpace(account.Name)) == 0 {
		return "The name must be informed"
	}
	if !collections.Contains(account.Tags, "balanceSheet") && !collections.Contains(account.Tags, "incomeStatement") {
		return "The financial statement must be informed"
	}
	if collections.Contains(account.Tags, "balanceSheet") && collections.Contains(account.Tags, "incomeStatement") {
		return "The statement must be either balance sheet or income statement"
	}
	if !collections.Contains(account.Tags, "debitBalance") && !collections.Contains(account.Tags, "creditBalance") {
		return "The normal balance must be informed"
	}
	if collections.Contains(account.Tags, "debitBalance") && collections.Contains(account.Tags, "creditBalance") {
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
	coaKey, err := db.DecodeKey(param["coa"])
	if err != nil {
		return err.Error()
	}
	if account.Key.IsZero() {
		if key, err := accountKeyWithNumber(db, context.Context{nil, nil}, account.Number,
			param["coa"]); err != nil {
			return err.Error()
		} else if !key.IsZero() {
			return "An account with this number already exists"
		}
	}
	if !account.Parent.IsZero() {
		var parent Account
		if _, err := db.Get(&parent, account.Parent.Encode()); err != nil {
			return err.Error()
		}
		if !strings.HasPrefix(account.Number, parent.Number) {
			return "The number must start with parent's number"
		}
		for key, value := range inheritedProperties {
			if collections.Contains(parent.Tags, key) && !collections.Contains(account.Tags, key) {
				return "The " + value + " must be same as the parent"
			}
		}
		if account.Parent.Parent().String() != coaKey.String() {
			return "The account's parent must belong to the same chart of accounts of the account"
		}
	}
	return ""
}

func (account *Account) Debit(value float64) float64 {
	if collections.Contains(account.Tags, "debitBalance") {
		return value
	} else {
		return -value
	}
}

func (account *Account) Credit(value float64) float64 {
	if collections.Contains(account.Tags, "creditBalance") {
		return value
	} else {
		return -value
	}
}

func (account *Account) String() string {
	return "Key: " + account.Key.String() + " Name: " + account.Name
}

type Transaction struct {
	db.Identifiable
	Debits               []Entry      `json:"debits"`
	Credits              []Entry      `json:"credits"`
	Date                 time.Time    `json:"date"`
	Memo                 string       `json:"memo"`
	Tags                 []string     `json:"tags"`
	User                 core.UserKey `json:"user"`
	AsOf                 time.Time    `json:"timestamp"`
	AccountsKeysAsString []string     `json:"-"`
	Moment               int64        `datastore:"-" json:"-"`
	Key_                 interface{}  `datastore:"-" json:"_id"`
}

type Entry struct {
	Account db.CKey `json:"account"`
	Value   float64 `json:"value"`
}

type transactionMetadata struct {
	Memo    string
	Tags    []string
	User    core.UserKey
	Removes int64
}

func (transaction *Transaction) ValidationMessage(db db.Db, param map[string]string) string {
	if len(transaction.Debits) == 0 {
		return "At least one debit must be informed"
	}
	if len(transaction.Credits) == 0 {
		return "At least one credit must be informed"
	}
	if transaction.Date.IsZero() {
		return "The date must be informed"
	}
	if len(strings.TrimSpace(transaction.Memo)) == 0 {
		return "The memo must be informed"
	}
	ev := func(arr []Entry) (string, float64) {
		sum := 0.0
		for _, e := range arr {
			if m := e.ValidationMessage(db, param); len(m) > 0 {
				return m, 0.0
			}
			sum += e.Value
		}
		return "", sum
	}
	var debitsSum, creditsSum float64
	var m string
	if m, debitsSum = ev(transaction.Debits); len(m) > 0 {
		return m
	}
	if m, creditsSum = ev(transaction.Credits); len(m) > 0 {
		return m
	}
	if xmath.Round(debitsSum*100) != xmath.Round(creditsSum*100) {
		return "The sum of debit values must be equals to the sum of credit values"
	}
	return ""
}

func (transaction *Transaction) SetDebitsAndCredits(debits, credits []Entry) {
	transaction.Debits = debits
	transaction.Credits = credits
}

func (transaction *Transaction) updateAccountsKeysAsString() {
	accountsKeysAsString := []string{}
	f := func(entries []Entry) {
		for _, e := range entries {
			accountsKeysAsString = append(accountsKeysAsString, e.Account.Encode())
		}
	}
	f(transaction.Debits)
	f(transaction.Credits)
	transaction.AccountsKeysAsString = accountsKeysAsString
}

func (transaction *Transaction) incrementValue(lookupAccount func(db.Key) *Account,
	addValue func(db.Key, float64)) {
	f := func(entries []Entry, f func(*Account, float64) float64) {
		for _, e := range entries {
			var accountKey db.Key
			accountKey, value := e.Account, e.Value
			for accountKey != nil && !accountKey.IsZero() {
				account := lookupAccount(accountKey)
				if account != nil {
					addValue(accountKey, f(account, value))
					accountKey = account.Parent
				} else {
					accountKey = nil
				}
			}
		}
	}
	f(transaction.Debits, (*Account).Debit)
	f(transaction.Credits, (*Account).Credit)
}

func (entry *Entry) ValidationMessage(db db.Db, param map[string]string) string {
	if entry.Account.IsZero() {
		return "The account must be informed for each entry"
	}
	var account = new(Account)
	if _, err := db.Get(account, entry.Account.Encode()); err != nil {
		return err.Error()
	}
	if account == nil {
		return "Account not found"
	}
	if !collections.Contains(account.Tags, "analytic") {
		return "The account must be analytic"
	}
	coaKey, err := db.DecodeKey(param["coa"])
	if err != nil {
		return err.Error()
	}
	if entry.Account.Parent().String() != coaKey.String() {
		return "The account must belong to the same chart of accounts of the transaction"
	}

	return ""
}

func AllChartsOfAccounts(c context.Context, m map[string]interface{}, _ map[string]string,
	_ core.UserKey) (interface{}, error) {
	_, chartsOfAccounts, err := c.Db.GetAll("ChartOfAccounts", "", &[]ChartOfAccounts{}, nil,
		[]string{"Name"})
	return chartsOfAccounts, err
}

func SaveChartOfAccounts(c context.Context, m map[string]interface{}, param map[string]string,
	userKey core.UserKey) (interface{}, error) {
	coa := &ChartOfAccounts{
		Name: m["name"].(string),
		User: userKey,
		AsOf: time.Now()}
	if coaKeyAsString, ok := param["coa"]; ok {
		if k, err := c.Db.DecodeKey(coaKeyAsString); err != nil {
			return nil, err
		} else {
			coa.SetKey(k)
		}
		var coa2 ChartOfAccounts
		if _, err := c.Db.Get(&coa2, param["coa"]); err != nil {
			return nil, err
		}
		coa.Space = coa2.Space
	} else {
		if k, err := c.Db.DecodeKey(param["space"]); err != nil {
			return nil, err
		} else {
			coa.Space = k.(db.CKey)
		}
	}
	_, err := c.Db.Save(coa, "ChartOfAccounts", "", param)
	if err != nil {
		return nil, err
	}
	err = c.Cache.Delete("ChartOfAccounts")
	return coa, err
}

func AllAccounts(c context.Context, m map[string]interface{}, param map[string]string,
	_ core.UserKey) (interface{}, error) {
	_, accounts, err := c.Db.GetAll("Account", param["coa"], &[]Account{}, nil, []string{"Number"})
	aa := accounts.(*[]Account)
	result := make([]Account, 0, len(*aa))
	for _, a := range *aa {
		if !a.Removed {
			result = result[0 : len(result)+1]
			result[len(result)-1] = a
		}
	}
	return result, err
}

func GetAccount(c context.Context, m map[string]interface{}, param map[string]string,
	_ core.UserKey) (result interface{}, err error) {
	if a, err := c.Db.Get(&Account{}, param["account"]); err != nil {
		return nil, err
	} else {
		if a.(*Account).Removed {
			return nil, fmt.Errorf("Account not found")
		}
		return a, nil
	}

}

func SaveAccount(c context.Context, m map[string]interface{}, param map[string]string,
	userKey core.UserKey) (item interface{}, err error) {

	account := &Account{
		Number:  m["number"].(string),
		Name:    m["name"].(string),
		Tags:    []string{},
		User:    userKey,
		AsOf:    time.Now(),
		Created: time.Now(),
		Removed: false}

	isUpdate := false

	if accountKeyAsString, ok := param["account"]; ok {
		isUpdate = true
		if k, err := c.Db.DecodeKey(accountKeyAsString); err != nil {
			return nil, err
		} else {
			account.SetKey(k)
		}
	}

	parent := &Account{}
	if isUpdate {
		var a Account
		if _, err = c.Db.Get(&a, account.Key.Encode()); err != nil {
			return
		}
		if !a.Parent.IsZero() {
			if _, err = c.Db.Get(parent, a.Parent.Encode()); err != nil {
				return
			}
		}
		account.Parent = a.Parent
		account.Number = a.Number
		account.Created = a.Created
	}
	if parentNumber, ok := m["parent"]; ok {
		var accounts []Account
		keys, _, err := c.Db.GetAll("Account", param["coa"], &accounts,
			db.M{"Number = ": parentNumber}, nil)
		if err != nil {
			return nil, err
		}
		if keys.Len() == 0 {
			return nil, fmt.Errorf("Parent not found: %v", parentNumber)
		}
		account.Parent = keys.KeyAt(0).(db.CKey)
		parent = &accounts[0]
		delete(m, "parent")
	}

	var retainedEarningsAccount bool
	for k, _ := range m {
		if k != "name" && k != "number" && k != "parent" {
			if k == "retainedEarnings" {
				retainedEarningsAccount = true
			} else if !collections.Contains(account.Tags, k) {
				_, ok1 := inheritedProperties[k]
				_, ok2 := nonInheritedProperties[k]
				if ok1 || ok2 {
					account.Tags = append(account.Tags, k)
				}
			}
		}
	}
	if !collections.Contains(account.Tags, "analytic") && !isUpdate {
		account.Tags = append(account.Tags, "analytic")
	}

	err = c.Db.Execute(func(tdb db.Db) (err error) {

		accountKey, err := tdb.Save(account, "Account", param["coa"], param)
		if err != nil {
			return
		}

		if retainedEarningsAccount {
			coa := new(ChartOfAccounts)
			if _, err = tdb.Get(coa, param["coa"]); err != nil {
				return
			}
			coa.RetainedEarningsAccount = accountKey.(db.CKey)
			if _, err = tdb.Save(coa, "ChartOfAccounts", "", param); err != nil {
				return
			}
		}

		if !account.Parent.IsZero() && !isUpdate {
			changed := false
			i := collections.IndexOf(parent.Tags, "analytic")
			if i != -1 {
				parent.Tags = append(parent.Tags[:i], parent.Tags[i+1:]...)
				changed = true
			}
			if !collections.Contains(parent.Tags, "synthetic") {
				parent.Tags = append(parent.Tags, "synthetic")
				changed = true
			}
			if changed {
				if _, err = tdb.Save(parent, "Account", param["coa"], param); err != nil {
					return
				}
			}
		}
		return
	})
	if err != nil {
		return
	}

	err = c.Cache.Delete("accounts_" + param["coa"])

	item = account
	return
}

func DeleteAccount(c context.Context, m map[string]interface{}, param map[string]string,
	userKey core.UserKey) (_ interface{}, err error) {

	key, err := c.Db.DecodeKey(param["account"])
	if err != nil {
		return
	}

	coaKey, err := c.Db.DecodeKey(param["coa"])
	if err != nil {
		return
	}

	checkReferences := func(kind, property, errorMessage string) error {
		if keys, _, err := c.Db.GetAllWithLimit(kind, param["coa"], nil,
			db.M{property: key}, nil, 1); err != nil {
			return err
		} else {
			if keys.Len() > 0 {
				return fmt.Errorf(errorMessage)
			}
		}
		return nil
	}

	err = checkReferences("Account", "Parent = ", "Child accounts found")
	if err != nil {
		return
	}
	err = checkReferences("Transaction", "Debits.Account = ",
		"Transactions referencing this account was found")
	if err != nil {
		return
	}
	err = checkReferences("Transaction", "Credits.Account = ",
		"Transactions referencing this account was found")
	if err != nil {
		return
	}
	/*
		if err = c.Db.Delete(key); err != nil {
			return
		}
	*/
	err = c.Db.Execute(func(tdb db.Db) error {
		var a Account
		if _, err := tdb.Get(&a, key.Encode()); err != nil {
			return err
		}
		a.Removed = true
		if _, err := tdb.Save(&a, "Account", coaKey.Encode(), param); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return
	}

	err = c.Cache.Delete("accounts_" + coaKey.Encode())

	return

}

func AllTransactions(c context.Context, m map[string]interface{}, param map[string]string,
	_ core.UserKey) (interface{}, error) {
	_, transactions, err := c.Db.GetAll("Transaction", param["coa"], &[]Transaction{}, nil,
		[]string{"Date", "AsOf"})
	return transactions, err
}

func GetTransaction(c context.Context, m map[string]interface{}, param map[string]string,
	_ core.UserKey) (result interface{}, err error) {
	space, ok := m["space"].(deb.Space)
	if !ok {
		return c.Db.Get(&Transaction{}, param["transaction"])
	} else {
		var moment deb.Moment
		if key, err := strconv.ParseUint(param["transaction"], 10, 64); err != nil {
			return nil, err
		} else {
			moment = deb.Moment(key)
		}
		space, err := space.Slice(nil, nil,
			[]deb.MomentRange{deb.MomentRange{Start: moment, End: moment}})
		if err != nil {
			return nil, err
		}
		accountKeys, accounts, err := Accounts(c, param["coa"], nil)
		if err != nil {
			return nil, err
		}
		transactions, keys, err := TransactionsFromSpace(space, accounts, accountKeys)
		if err != nil {
			return nil, err
		}
		transactions[0].Key_ = keys[0]
		return transactions[0], nil
	}
}

func SaveTransaction(c context.Context, maps []map[string]interface{}, param map[string]string,
	userKey core.UserKey) (item interface{}, err error) {

	if len(maps) == 0 {
		return nil, fmt.Errorf("maps is empty")
	}

	if len(maps) > 1 {
		return SaveTransactions(c, maps, param, userKey)
	}

	m := maps[0]

	asOf := time.Now()
	transaction := &Transaction{
		Memo: m["memo"].(string),
		AsOf: asOf,
		User: userKey}
	transaction.Date, err = time.Parse(time.RFC3339, m["date"].(string))
	if err != nil {
		return
	}

	coaKey, err := c.Db.DecodeKey(param["coa"])
	if err != nil {
		return
	}

	entriesArray := func(entriesMapArray []interface{}) (result []Entry, err error) {
		result = make([]Entry, len(entriesMapArray))
		for i := 0; i < len(entriesMapArray); i++ {
			entryMap := entriesMapArray[i].(map[string]interface{})
			var (
				key db.Key
				err error
			)
			if am, ok := m["accounts_map"]; ok {
				key = am.(map[string]db.Key)[entryMap["account"].(string)]
			} else {
				key, err = accountKeyWithNumber(nil, c, entryMap["account"].(string),
					param["coa"])
			}
			if err != nil {
				return nil, err
			} else if key.IsZero() {
				return nil, fmt.Errorf("Account '%v' not found", entryMap["account"])
			} else {
				result[i] = Entry{
					Account: key.(db.CKey),
					Value:   xmath.Round(entryMap["value"].(float64)*100) / 100}
			}
		}
		return
	}
	if debits, err := entriesArray(m["debits"].([]interface{})); err != nil {
		return nil, err
	} else if credits, err := entriesArray(m["credits"].([]interface{})); err != nil {
		return nil, err
	} else {
		transaction.SetDebitsAndCredits(debits, credits)
	}

	isUpdate := false
	if _, ok := param["transaction"]; ok {
		isUpdate = true
	}

	transaction.updateAccountsKeysAsString()

	space, ok := m["space"].(deb.Space)
	if !ok {
		if isUpdate {
			if k, err := c.Db.DecodeKey(param["transaction"]); err != nil {
				return nil, err
			} else {
				transaction.SetKey(k)
			}
		}
		var transactionKey db.Key
		if k, err := c.Db.Save(transaction, "Transaction", param["coa"], param); err != nil {
			return nil, err
		} else {
			transactionKey = k
		}
		if isUpdate {
			if err = c.Cache.Delete("transactions_asof_" + coaKey.Encode()); err != nil {
				return nil, err
			}
			if err = c.Cache.Delete("balances_asof_" + coaKey.Encode()); err != nil {
				return nil, err
			}
			err = nil
		} else {
			if err = c.Cache.Set("transactions_asof_"+coaKey.Encode(), asOf); err != nil {
				return nil, err
			}
		}
		err = c.Cache.Delete("transactions_" + coaKey.Encode())
		transaction.SetKey(transactionKey)
	} else {
		if isUpdate {
			DeleteTransaction(c, m, param, userKey)
		}
		accounts, _ := m["accounts_sorted_by_creation"].([]*Account)
		accountsKeys, _ := m["accounts_keys_sorted_by_creation"].(db.Keys)
		err = appendTransactionOnSpace(c, coaKey.Encode(), space, transaction, -1,
			accounts, accountsKeys)
	}

	item = transaction

	return
}

func SaveTransactions(c context.Context, maps []map[string]interface{}, param map[string]string,
	userKey core.UserKey) (item interface{}, err error) {
	if len(maps) == 0 {
		return nil, nil
	}
	space, ok := maps[0]["space"].(deb.Space)
	if !ok {
		return nil, fmt.Errorf("Space not informed")
	}
	now := time.Now().UnixNano()
	transactions := make([]*deb.Transaction, len(maps))
	_, accounts, err := accountsSortedByCreation(c, param["coa"])
	if err != nil {
		return nil, err
	}
	accountsMap := map[string]int{}
	for i, a := range accounts {
		accountsMap[a.Number] = i + 1
	}
	for i, m := range maps {
		date, err := time.Parse(time.RFC3339, m["date"].(string))
		if err != nil {
			return nil, err
		}
		entries := deb.Entries{}
		addEntry := func(e interface{}, signal int) {
			em := e.(map[string]interface{})
			entries[deb.Account(accountsMap[em["account"].(string)])] =
				int64(signal) * int64(xmath.Round(em["value"].(float64)*100))
		}
		for _, e := range m["debits"].([]interface{}) {
			addEntry(e, 1)
		}
		for _, e := range m["credits"].([]interface{}) {
			addEntry(e, -1)
		}
		memo, ok := m["memo"].(string)
		if !ok {
			return nil, fmt.Errorf("Memo must be informed")
		}
		metadata := transactionMetadata{memo, nil, userKey, -1}
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		if err := enc.Encode(metadata); err != nil {
			return nil, err
		}
		transactions[i] = &deb.Transaction{
			Moment:   deb.Moment(now + i),
			Date:     SerializedDate(date),
			Entries:  entries,
			Metadata: buf.Bytes()}
	}
	ch := make(chan *deb.Transaction)
	go func() {
		for _, t := range transactions {
			log.Println(t)
			ch <- t
		}
		close(ch)
	}()
	return nil, space.Append(deb.ChannelSpace(ch))
}

func appendTransactionOnSpace(c context.Context, coaKey string, space deb.Space,
	transaction *Transaction, removes int64, accounts []*Account, accountKeys db.Keys) error {
	if accounts == nil {
		var err error
		accountKeys, accounts, err = accountsSortedByCreation(c, coaKey)
		if err != nil {
			return err
		}
	}
	values := make([]int64, len(accounts))
	for _, e := range transaction.Debits {
		found := false
		for i, k := range accountKeys {
			if k.Encode() == e.Account.Encode() {
				values[i] += int64(xmath.Round(e.Value * 100))
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("Account %v not found", e.Account.Encode())
		}
	}
	for _, e := range transaction.Credits {
		found := false
		for i, k := range accountKeys {
			if k.Encode() == e.Account.Encode() {
				values[i] += -int64(xmath.Round(e.Value * 100))
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("Account %v not found", e.Account.Encode())
		}
	}
	dateOffset := SerializedDate(transaction.Date) - 1
	metadata := transactionMetadata{transaction.Memo, transaction.Tags, transaction.User, removes}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(metadata); err != nil {
		return err
	}
	return space.Append(deb.NewSmallSpaceWithOffset(deb.Array{{values}}.Transposed(),
		uint64(dateOffset), uint64(transaction.AsOf.UnixNano()), [][][]byte{{buf.Bytes()}}))
}

func accountsSortedByCreation(c context.Context, coaKey string) (db.Keys, []*Account, error) {
	var accounts []*Account
	keys, _, err := c.Db.GetAllFromCache("Account", coaKey, &accounts, nil,
		[]string{"Created"}, c.Cache, "accounts_"+coaKey)
	if err != nil {
		return nil, nil, err
	}
	return keys, accounts, nil
}

func DeleteTransaction(c context.Context, m map[string]interface{}, param map[string]string,
	userKey core.UserKey) (_ interface{}, err error) {

	space, ok := m["space"].(deb.Space)
	if !ok {
		key, err := c.Db.DecodeKey(param["transaction"])
		if err != nil {
			return nil, err
		}
		if err = c.Db.Delete(key); err != nil {
			return nil, err
		}
		if err = c.Cache.Delete("transactions_asof_" + key.Parent().Encode()); err != nil {
			return nil, err
		}
		if err = c.Cache.Delete("balances_asof_" + key.Parent().Encode()); err != nil {
			return nil, err
		}
		if err = c.Cache.Delete("transactions_" + key.Parent().Encode()); err != nil {
			return nil, err
		}

		err = nil
	} else {
		t, err := GetTransaction(c, m, param, userKey)
		if err != nil {
			return nil, err
		}
		tx := t.(*Transaction)
		deb := make([]Entry, len(tx.Credits))
		cre := make([]Entry, len(tx.Debits))
		for i, e := range tx.Debits {
			cre[i] = Entry{Account: e.Account, Value: e.Value}
		}
		for i, e := range tx.Credits {
			deb[i] = Entry{Account: e.Account, Value: e.Value}
		}
		tx.Debits = deb
		tx.Credits = cre
		var removes int64
		if removes, err = strconv.ParseInt(param["transaction"], 10, 64); err != nil {
			return nil, err
		}

		appendTransactionOnSpace(c, param["coa"], space, tx, removes, nil, nil)
	}

	return

}

type byCreation struct {
	a []*Account
	k db.Keys
}

func (a byCreation) Len() int           { return len(a.a) }
func (a byCreation) Swap(i, j int)      { a.a[i], a.a[j] = a.a[j], a.a[i]; a.k[i], a.k[j] = a.k[j], a.k[i] }
func (a byCreation) Less(i, j int) bool { return a.a[i].Created.Before(a.a[j].Created) }

func AccountsByCreation(accounts []*Account, keys db.Keys) ([]*Account, db.Keys) {
	sortedAccounts := make([]*Account, len(accounts))
	sortedKeys := make(db.Keys, len(keys))
	copy(sortedAccounts, accounts)
	copy(sortedKeys, keys)
	sort.Sort(byCreation{sortedAccounts, sortedKeys})
	return sortedAccounts, sortedKeys
}

func NewTransactionFromSpace(t *deb.Transaction, keys db.Keys) (*Transaction,
	*transactionMetadata, error) {
	d := time.Date(int(t.Date%100000000/10000), time.Month(t.Date%10000/100), int(t.Date%100),
		0, 0, 0, 0, time.UTC)
	m := time.Unix(0, int64(t.Moment))
	buf := bytes.NewBuffer(t.Metadata)
	dec := gob.NewDecoder(buf)
	var tm transactionMetadata
	if err := dec.Decode(&tm); err != nil {
		return nil, nil, err
	}
	deb := []Entry{}
	cre := []Entry{}
	for k, v := range t.Entries {
		if v > 0 {
			deb = append(deb, Entry{Account: keys[k-1], Value: float64(v) / 100})
		} else {
			cre = append(cre, Entry{Account: keys[k-1], Value: -float64(v) / 100})
		}
	}
	transaction := &Transaction{Date: d, AsOf: m, Debits: deb, Credits: cre,
		Memo: tm.Memo, Tags: tm.Tags, User: tm.User, Moment: int64(t.Moment)}
	return transaction, &tm, nil
}

func TransactionsFromSpace(space deb.Space, accounts []*Account,
	accountKeys db.Keys) ([]*Transaction, []interface{}, error) {
	_, sortedKeys := AccountsByCreation(accounts, accountKeys)
	ch, errc := space.Transactions()
	transactions := []*Transaction{}
	var (
		err error
		md  *transactionMetadata
	)
	for t := range ch {
		if err != nil {
			continue
		}
		var tx *Transaction
		tx, md, err = NewTransactionFromSpace(t, sortedKeys)
		if md.Removes == -1 {
			transactions = append(transactions, tx)
		} else {
			for i, t := range transactions {
				if t.Moment == md.Removes {
					transactions = append(transactions[:i], transactions[i+1:]...)
					break
				}
			}
		}
	}
	if err != nil {
		return nil, nil, err
	}
	if err = <-errc; err != nil {
		return nil, nil, err
	}
	sort.Sort(ByDateAndAsOf(transactions))
	transactionKeys := make([]interface{}, len(transactions))
	for i, t := range transactions {
		transactionKeys[i] = fmt.Sprintf("%v", t.Moment)
	}
	return transactions, transactionKeys, nil
}

func SerializedDate(d time.Time) deb.Date {
	return deb.Date(d.Year()*10000 + int(d.Month())*100 + d.Day())
}

func Accounts(c context.Context, coaKey string, filters map[string]interface{}) (keys db.Keys,
	accounts []*Account, err error) {
	if filters == nil {
		filters = map[string]interface{}{}
	}
	filters["Removed ="] = false
	keys, _, err = c.Db.GetAllFromCache("Account", coaKey, &accounts, filters, []string{"Number"},
		c.Cache, "accounts_"+coaKey)
	if err != nil {
		return
	}
	return
}

func Transactions(c context.Context, coaKey string, filters map[string]interface{}) (keys db.Keys,
	transactions []*Transaction, err error) {
	keys, _, err = c.Db.GetAll("Transaction", coaKey, &transactions, filters,
		[]string{"Date", "AsOf"})
	if err != nil {
		return
	}
	return
}

type TransactionWithValue struct {
	Transaction
	Value float64
	Key   interface{}
}

func TransactionsWithValue(c context.Context, coaKey string, account *Account, from,
	to time.Time) (transactionsWithValue []*TransactionWithValue, balance float64, err error) {

	b, err := Balances(c, coaKey, time.Date(1000, 1, 1, 0, 0, 0, 0, time.UTC),
		from.AddDate(0, 0, -1), map[string]interface{}{"Number =": account.Number})
	if err != nil {
		return
	}
	balance = 0.0
	if len(b) > 0 {
		balance = b[0]["value"].(float64)
	}

	var dbkeys db.Keys
	var transactions []*Transaction
	if dbkeys, _, err = c.Db.GetAll("Transaction", coaKey, &transactions,
		db.M{"AccountsKeysAsString =": account.Key.Encode(), "Date >=": from, "Date <=": to},
		[]string{"Date", "AsOf"}); err != nil {
		return
	}
	keys := make([]interface{}, len(dbkeys))
	for i, k := range dbkeys {
		keys[i] = k
	}
	transactionsWithValue = TransactionsWithValueFromTransactions(transactions, keys, account)
	return
}

func TransactionsWithValueFromTransactions(transactions []*Transaction, keys []interface{},
	account *Account) []*TransactionWithValue {
	var t *TransactionWithValue
	lookupAccount := func(key db.Key) *Account {
		if key.String() == account.Key.String() {
			return account
		}
		return nil
	}
	addValue := func(key db.Key, value float64) {
		t.Value += value
	}
	transactionsWithValue := []*TransactionWithValue{}
	for i, t_ := range transactions {
		t = &TransactionWithValue{*t_, 0, keys[i]}
		t.incrementValue(lookupAccount, addValue)
		transactionsWithValue = append(transactionsWithValue, t)
	}
	return transactionsWithValue
}

type ByDateAndAsOf []*Transaction

func (a ByDateAndAsOf) Len() int      { return len(a) }
func (a ByDateAndAsOf) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByDateAndAsOf) Less(i, j int) bool {
	return a[i].Date.Before(a[j].Date) ||
		(a[i].Date.Equal(a[j].Date) && a[i].AsOf.Before(a[j].AsOf))
}

func Balances(c context.Context, coaKey string, from, to time.Time, accountFilters db.M) (result []db.M, err error) {

	var transactionsAsOf, balancesAsOf time.Time

	err = c.Cache.Get("transactions_asof_"+coaKey, &transactionsAsOf)
	if err != nil {
		return
	}
	if transactionsAsOf.IsZero() {
		var transactions []*Transaction
		_, _, err = c.Db.GetAllWithLimit("Transaction", coaKey, &transactions, nil, []string{"-AsOf"}, 1)
		if err != nil {
			return nil, err
		}
		if len(transactions) == 1 {
			transactionsAsOf = transactions[0].AsOf
			if err = c.Cache.Set("transactions_asof_"+coaKey, transactionsAsOf); err != nil {
				return nil, err
			}
		}
	}

	timespanAsString := from.String() + "_" + to.String()

	var balancesAsOfMap map[string]time.Time
	err = c.Cache.Get("balances_asof_"+coaKey, &balancesAsOfMap)
	if err != nil {
		return
	} else if balancesAsOfMap != nil {
		balancesAsOf = balancesAsOfMap[timespanAsString]
	}

	err = c.Cache.Get("balances_"+coaKey+"_"+timespanAsString, &result)

	if (result == nil || len(result) == 0) ||
		(err == nil && (transactionsAsOf != balancesAsOf || balancesAsOf.IsZero())) {

		accountKeys, accounts, err := Accounts(c, coaKey, nil)
		if err != nil {
			return nil, err
		}

		resultMap := map[string]map[string]interface{}{}

		filter := db.M{}

		if transactionsAsOf != balancesAsOf &&
			!transactionsAsOf.IsZero() && !balancesAsOf.IsZero() {
			filter["AsOf >"] = balancesAsOf
			filter["AsOf <="] = transactionsAsOf
			for _, item := range result {
				resultMap[item["account"].(*Account).Key.String()] = item
			}
		} else {
			filter["Date >="] = from
			filter["Date <="] = to
			result = []db.M{}
			for i, a := range accounts {
				a.SetKey(accountKeys.KeyAt(i))
				item := db.M{"account": a, "value": 0.0}
				result = append(result, item)
				resultMap[accountKeys.KeyAt(i).String()] = item
			}
		}

		var transactions []*Transaction
		if _, _, err = c.Db.GetAll("Transaction", coaKey, &transactions, filter, nil); err != nil {
			return nil, err
		}

		sort.Sort(ByDateAndAsOf(transactions))

		lookupAccount := func(key db.Key) *Account {
			if key.IsZero() {
				return nil
			}
			item := resultMap[key.String()]
			if item == nil || item["account"] == nil {
				return nil
			} else {
				return item["account"].(*Account)
			}
		}
		addValue := func(key db.Key, value float64) {
			item := resultMap[key.String()]
			item["value"] = item["value"].(float64) + value
			item["value"] = xmath.Round(item["value"].(float64)*100) / 100
		}
		for _, t := range transactions {
			if !t.Date.Before(from) && !t.Date.After(to) {
				t.incrementValue(lookupAccount, addValue)
			}
		}

		if err = c.Cache.Set("balances_"+coaKey+"_"+timespanAsString, result); err != nil {
			return nil, err
		}
		if balancesAsOfMap == nil {
			balancesAsOfMap = map[string]time.Time{}
		}
		balancesAsOfMap[timespanAsString] = transactionsAsOf
		if err = c.Cache.Set("balances_asof_"+coaKey, balancesAsOfMap); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	if accountFilters != nil {
		filteredResult := []db.M{}
		for _, item := range result {
			if ok, err := db.Matches(item["account"].(*Account), accountFilters); err != nil {
				return nil, err
			} else if ok {
				filteredResult = append(filteredResult, item)
			}
		}
		result = filteredResult
	}

	return result, nil
}

func accountKeyWithNumber(d db.Db, c context.Context, number, coa string) (db.Key, error) {
	var (
		keys db.Keys
		err  error
	)
	if d != nil {
		keys, _, err = d.GetAll("Account", coa, nil,
			db.M{"Number = ": number, "Removed = ": false}, nil)
	} else {
		keys, _, err = Accounts(c, coa, db.M{"Number = ": number, "Removed = ": false})
	}
	if err != nil {
		return d.NewKey(), err
	} else if keys.Len() == 0 {
		return d.NewKey(), nil
	} else {
		return keys.KeyAt(0), nil
	}
}

func Migrate(c context.Context, coa *ChartOfAccounts, coaKey, coa2Key string, space deb.Space,
	key db.Key, userKey core.UserKey) (interface{}, error) {

	var (
		coa2     *ChartOfAccounts
		accounts *[]*Account
	)

	am := map[string]*Account{}
	am2 := map[string]db.Key{}
	param := map[string]string{}

	if ak, aa, err := c.Db.GetAll("Account", coaKey, &[]*Account{}, nil,
		[]string{"Number"}); err != nil {
		return nil, err
	} else {
		accounts = aa.(*[]*Account)
		for i, a := range *accounts {
			am[ak[i].Encode()] = a
		}
	}

	if coa2Key == "" {
		coa2 = &ChartOfAccounts{
			Name:  coa.Name + "/2",
			Space: key.(db.CKey),
			User:  userKey,
			AsOf:  time.Now()}

		_, err := c.Db.Save(coa2, "ChartOfAccounts", "", nil)
		if err != nil {
			return nil, err
		}
		err = c.Cache.Delete("ChartOfAccounts")
		param["coa"] = coa2.Key.Encode()
		for _, a := range *accounts {
			if a.Removed {
				continue
			}
			m := map[string]interface{}{}
			m["name"] = a.Name
			m["number"] = a.Number
			if !a.Parent.IsZero() {
				m["parent"] = am[a.Parent.Encode()].Number
			}
			if a.Tags != nil {
				for _, t := range a.Tags {
					m[t] = true
				}
			}
			if account, err := SaveAccount(c, m, param, userKey); err != nil {
				return nil, err
			} else {
				am2[a.Number] = account.(*Account).GetKey()
			}
		}
	} else {
		param["coa"] = coa2Key
		coa2 = &ChartOfAccounts{}
		if _, err := c.Db.Get(coa2, coa2Key); err != nil {
			return nil, err
		}
		ak2, aa2, err := c.Db.GetAll("Account", coa2Key, &[]*Account{}, nil, []string{"Number"})
		if err != nil {
			return nil, err
		}
		accounts2 := aa2.(*[]*Account)
		for i, a := range *accounts2 {
			am2[a.Number] = ak2[i]
		}
	}
	_, tt, err := c.Db.GetAll("Transaction", coaKey, &[]*Transaction{}, nil, []string{"AsOf"})
	if err != nil {
		return nil, err
	}
	transactions := tt.(*[]*Transaction)
	accountsSortedKeys, accountsSorted, err := accountsSortedByCreation(c, coa2Key)
	if err != nil {
		return nil, err
	}
	maps := []map[string]interface{}{}
	for _, t := range *transactions {
		m := map[string]interface{}{}
		debits := make([]interface{}, len(t.Debits))
		credits := make([]interface{}, len(t.Credits))
		for i, e := range t.Debits {
			m := map[string]interface{}{}
			m["account"] = am[e.Account.Encode()].Number
			m["value"] = e.Value
			debits[i] = m
		}
		for i, e := range t.Credits {
			m := map[string]interface{}{}
			m["account"] = am[e.Account.Encode()].Number
			m["value"] = e.Value
			credits[i] = m
		}
		m["memo"] = t.Memo
		m["date"] = t.Date.Format(time.RFC3339)
		m["debits"] = debits
		m["credits"] = credits
		m["space"] = space
		m["accounts_map"] = am2
		m["accounts_sorted_by_creation"] = accountsSorted
		m["accounts_keys_sorted_by_creation"] = accountsSortedKeys
		maps = append(maps, m)
	}
	if _, err := SaveTransactions(c, maps, param, userKey); err != nil {
		return nil, err
	}
	return fmt.Sprintf("Migrated to %v. %v accounts", coa2.Name, len(*accounts)), nil
}
