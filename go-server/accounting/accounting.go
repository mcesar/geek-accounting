package accounting

import (
	//"log"
	//"runtime/debug"
	"appengine"
	"appengine/datastore"
	"appengine/memcache"
	//"encoding/json"
	"fmt"
	"github.com/mcesarhm/geek-accounting/go-server/core"
	"github.com/mcesarhm/geek-accounting/go-server/db"
	"github.com/mcesarhm/geek-accounting/go-server/util"
	"sort"
	//"strconv"
	"strings"
	"time"
)

func UpdateSchema(c appengine.Context) (err error) {
	q := datastore.NewQuery("Transaction")
	transactions := []*Transaction{}
	keys, err := q.GetAll(c, &transactions)
	if err != nil {
		return err
	}
	for i, tx := range transactions {
		if _, err := datastore.Put(c, keys[i], tx); err != nil {
			return err
		}
	}
	return nil
}

type ChartOfAccounts struct {
	db.Identifiable
	Name                    string       `json:"name"`
	RetainedEarningsAccount db.Key       `json:"retainedEarningsAccount"`
	User                    core.UserKey `json:"user"`
	AsOf                    time.Time    `json:"timestamp"`
}

func (coa *ChartOfAccounts) ValidationMessage(_ appengine.Context, _ map[string]string) string {
	if len(strings.TrimSpace(coa.Name)) == 0 {
		return "The name must be informed"
	}
	return ""
}

func (coa *ChartOfAccounts) Load(c <-chan datastore.Property) (err error) {
	f := make(chan datastore.Property, 32)
	go func() {
		for p := range c {
			if p.Name == "User" || p.Name == "RetainedEarningsAccount" {
				p.Name += ".DsKey"
			}
			f <- p
		}
		close(f)
	}()
	return datastore.LoadStruct(coa, f)
}

func (coa *ChartOfAccounts) Save(c chan<- datastore.Property) error {
	return datastore.SaveStruct(coa, c)
}

type Account struct {
	db.Identifiable
	Number string       `json:"number"`
	Name   string       `json:"name"`
	Tags   []string     `json:"tags"`
	Parent db.Key       `json:"parent"`
	User   core.UserKey `json:"user"`
	AsOf   time.Time    `json:"timestamp"`
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

func (account *Account) ValidationMessage(c appengine.Context, param map[string]string) string {
	if len(strings.TrimSpace(account.Number)) == 0 {
		return "The number must be informed"
	}
	if len(strings.TrimSpace(account.Name)) == 0 {
		return "The name must be informed"
	}
	if !util.Contains(account.Tags, "balanceSheet") && !util.Contains(account.Tags, "incomeStatement") {
		return "The financial statement must be informed"
	}
	if util.Contains(account.Tags, "balanceSheet") && util.Contains(account.Tags, "incomeStatement") {
		return "The statement must be either balance sheet or income statement"
	}
	if !util.Contains(account.Tags, "debitBalance") && !util.Contains(account.Tags, "creditBalance") {
		return "The normal balance must be informed"
	}
	if util.Contains(account.Tags, "debitBalance") && util.Contains(account.Tags, "creditBalance") {
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
	if account.Key.IsNil() {
		q := datastore.NewQuery("Account").Ancestor(coaKey.DsKey).Filter("Number = ", account.Number).KeysOnly()
		keys, err := q.GetAll(c, nil)
		if err != nil {
			return err.Error()
		}
		if len(keys) != 0 {
			return "An account with this number already exists"
		}
	}
	if !account.Parent.IsNil() {
		var parent Account
		if err := datastore.Get(c, account.Parent.DsKey, &parent); err != nil {
			return err.Error()
		}
		if !strings.HasPrefix(account.Number, parent.Number) {
			return "The number must start with parent's number"
		}
		for key, value := range inheritedProperties {
			if util.Contains(parent.Tags, key) && !util.Contains(account.Tags, key) {
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
	if util.Contains(account.Tags, "debitBalance") {
		return value
	} else {
		return -value
	}
}

func (account *Account) Credit(value float64) float64 {
	if util.Contains(account.Tags, "creditBalance") {
		return value
	} else {
		return -value
	}
}

func (account *Account) String() string {
	return "Key: " + account.Key.String() + " Name: " + account.Name
}

func (account *Account) Load(c <-chan datastore.Property) (err error) {
	f := make(chan datastore.Property, 32)
	go func() {
		for p := range c {
			if p.Name == "User" || p.Name == "Parent" {
				p.Name += ".DsKey"
			}
			f <- p
		}
		close(f)
	}()
	return datastore.LoadStruct(account, f)
}

func (account *Account) Save(c chan<- datastore.Property) error {
	return datastore.SaveStruct(account, c)
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
}

type Entry struct {
	Account db.Key  `json:"account"`
	Value   float64 `json:"value"`
}

func (transaction *Transaction) ValidationMessage(c appengine.Context, param map[string]string) string {
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
			if m := e.ValidationMessage(c, param); len(m) > 0 {
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
	if util.Round(debitsSum*100) != util.Round(creditsSum*100) {
		return "The sum of debit values must be equals to the sum of credit values"
	}
	return ""
}

func (transaction *Transaction) SetDebitsAndCredits(debits, credits []Entry) {
	transaction.Debits = debits
	transaction.Credits = credits
}

func (transaction *Transaction) Load(c <-chan datastore.Property) (err error) {
	f := make(chan datastore.Property, 32)
	go func() {
		for p := range c {
			if p.Name == "User" || p.Name == "Debits.Account" || p.Name == "Credits.Account" {
				p.Name += ".DsKey"
			}
			f <- p
		}
		close(f)
	}()
	return datastore.LoadStruct(transaction, f)
}

func (transaction *Transaction) Save(c chan<- datastore.Property) error {
	transaction.updateAccountsKeysAsString()
	return datastore.SaveStruct(transaction, c)
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

func (transaction *Transaction) incrementValue(lookupAccount func(db.Key) *Account, addValue func(db.Key, float64)) {
	f := func(entries []Entry, f func(*Account, float64) float64) {
		for _, e := range entries {
			accountKey, value := e.Account, e.Value
			for !accountKey.IsNil() {
				account := lookupAccount(accountKey)
				if account != nil {
					addValue(accountKey, f(account, value))
					accountKey = account.Parent
				} else {
					accountKey = db.NewNilKey()
				}
			}
		}
	}
	f(transaction.Debits, (*Account).Debit)
	f(transaction.Credits, (*Account).Credit)
}

func (entry *Entry) ValidationMessage(c appengine.Context, param map[string]string) string {
	if entry.Account.IsNil() {
		return "The account must be informed for each entry"
	}
	var account = new(Account)
	if err := datastore.Get(c, entry.Account.DsKey, account); err != nil {
		return err.Error()
	}
	if account == nil {
		return "Account not found"
	}
	if !util.Contains(account.Tags, "analytic") {
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

func AllChartsOfAccounts(c appengine.Context, _ map[string]string, _ core.UserKey) (interface{}, error) {
	_, chartsOfAccounts, err := db.GetAll(c, "ChartOfAccounts", "", &[]ChartOfAccounts{}, nil, []string{"Name"})
	return chartsOfAccounts, err
}

func SaveChartOfAccounts(c appengine.Context, m map[string]interface{}, param map[string]string, userKey core.UserKey) (interface{}, error) {
	coa := &ChartOfAccounts{
		Name: m["name"].(string),
		User: userKey,
		AsOf: time.Now()}
	_, err := db.Save(c, coa, "ChartOfAccounts", "", param)
	return coa, err
}

func AllAccounts(c appengine.Context, param map[string]string, _ core.UserKey) (interface{}, error) {
	_, accounts, err := db.GetAll(c, "Account", param["coa"], &[]Account{}, nil, []string{"Number"})
	return accounts, err
}

func GetAccount(c appengine.Context, param map[string]string, _ core.UserKey) (result interface{}, err error) {
	return db.Get(c, &Account{}, param["account"])
}

func SaveAccount(c appengine.Context, m map[string]interface{}, param map[string]string, userKey core.UserKey) (item interface{}, err error) {

	account := &Account{
		Number: m["number"].(string),
		Name:   m["name"].(string),
		Tags:   []string{},
		User:   userKey,
		AsOf:   time.Now()}

	isUpdate := false

	if accountKeyAsString, ok := param["account"]; ok {
		isUpdate = true
		account.Key, err = db.DecodeKey(accountKeyAsString)
		if err != nil {
			return
		}
	}

	coaKey, err := db.DecodeKey(param["coa"])
	if err != nil {
		return
	}

	parent := &Account{}
	if isUpdate {
		var a Account
		err = datastore.Get(c, account.Key.DsKey, &a)
		if err != nil {
			return
		}
		if !a.Parent.IsNil() {
			err = datastore.Get(c, a.Parent.DsKey, parent)
			if err != nil {
				return
			}
		}
		account.Parent = a.Parent
		account.Number = a.Number
	}
	if parentNumber, ok := m["parent"]; ok {
		q := datastore.NewQuery("Account").Ancestor(coaKey.DsKey).Filter("Number = ", parentNumber)
		var accounts []Account
		keys, err := q.GetAll(c, &accounts)
		if err != nil {
			return nil, err
		}
		if len(keys) == 0 {
			return nil, fmt.Errorf("Parent not found")
		}
		account.Parent = db.Key{keys[0]}
		parent = &accounts[0]
		delete(m, "parent")
	}

	var retainedEarningsAccount bool
	for k, _ := range m {
		if k != "name" && k != "number" && k != "number" {
			if k == "retainedEarnings" {
				retainedEarningsAccount = true
			} else if !util.Contains(account.Tags, k) {
				_, ok1 := inheritedProperties[k]
				_, ok2 := nonInheritedProperties[k]
				if ok1 || ok2 {
					account.Tags = append(account.Tags, k)
				}
			}
		}
	}
	if !util.Contains(account.Tags, "analytic") && !isUpdate {
		account.Tags = append(account.Tags, "analytic")
	}

	err = datastore.RunInTransaction(c, func(c appengine.Context) (err error) {

		accountKey, err := db.Save(c, account, "Account", param["coa"], param)
		if err != nil {
			return
		}

		if retainedEarningsAccount {
			coa := new(ChartOfAccounts)
			if err = datastore.Get(c, coaKey.DsKey, coa); err != nil {
				return
			}
			coa.RetainedEarningsAccount = accountKey
			if _, err = datastore.Put(c, coaKey.DsKey, coa); err != nil {
				return
			}
		}

		if !account.Parent.IsNil() && !isUpdate {
			i := util.IndexOf(parent.Tags, "analytic")
			if i != -1 {
				parent.Tags = append(parent.Tags[:i], parent.Tags[i+1:]...)
			}
			if !util.Contains(parent.Tags, "synthetic") {
				parent.Tags = append(parent.Tags, "synthetic")
			}
			if _, err = datastore.Put(c, account.Parent.DsKey, parent); err != nil {
				return
			}
		}
		return
	}, nil)
	if err != nil {
		return
	}

	if err = memcache.Delete(c, "accounts_"+coaKey.Encode()); err == memcache.ErrCacheMiss {
		err = nil
	}

	item = account
	return
}

func DeleteAccount(c appengine.Context, m map[string]interface{}, param map[string]string, userKey core.UserKey) (_ interface{}, err error) {

	key, err := db.DecodeKey(param["account"])
	if err != nil {
		return
	}

	coaKey, err := db.DecodeKey(param["coa"])
	if err != nil {
		return
	}

	checkReferences := func(kind, property, errorMessage string) error {
		q := datastore.NewQuery(kind).Ancestor(coaKey.DsKey).Filter(property, key).KeysOnly()
		var keys []*datastore.Key
		if keys, err = q.GetAll(c, nil); err != nil {
			return err
		}
		if len(keys) > 0 {
			return fmt.Errorf(errorMessage)
		}
		return nil
	}

	err = checkReferences("Account", "Parent = ", "Child accounts found")
	if err != nil {
		return
	}
	err = checkReferences("Transaction", "Debits.Account = ", "Transactions referencing this account was found")
	if err != nil {
		return
	}
	err = checkReferences("Transaction", "Credits.Account = ", "Transactions referencing this account was found")
	if err != nil {
		return
	}

	err = datastore.Delete(c, key.DsKey)
	if err != nil {
		return
	}

	if err = memcache.Delete(c, "accounts_"+coaKey.Encode()); err == memcache.ErrCacheMiss {
		err = nil
	}

	return

}

func AllTransactions(c appengine.Context, param map[string]string, _ core.UserKey) (interface{}, error) {
	_, transactions, err := db.GetAll(c, "Transaction", param["coa"], &[]Transaction{}, nil, []string{"Date", "AsOf"})
	return transactions, err
}

func GetTransaction(c appengine.Context, param map[string]string, _ core.UserKey) (result interface{}, err error) {
	return db.Get(c, &Transaction{}, param["transaction"])
}

func SaveTransaction(c appengine.Context, m map[string]interface{}, param map[string]string, userKey core.UserKey) (item interface{}, err error) {

	asOf := time.Now()

	transaction := &Transaction{
		Memo: m["memo"].(string),
		AsOf: asOf,
		User: userKey}
	transaction.Date, err = time.Parse(time.RFC3339, m["date"].(string))
	if err != nil {
		return
	}

	coaKey, err := db.DecodeKey(param["coa"])
	if err != nil {
		return
	}

	entriesArray := func(entriesMapArray []interface{}) (result []Entry, err error) {
		result = make([]Entry, len(entriesMapArray))
		for i := 0; i < len(entriesMapArray); i++ {
			entryMap := entriesMapArray[i].(map[string]interface{})
			q := datastore.NewQuery("Account").Ancestor(coaKey.DsKey).Filter("Number = ", entryMap["account"]).KeysOnly()
			var keys []*datastore.Key
			if keys, err = q.GetAll(c, nil); err != nil {
				return
			}
			if len(keys) == 0 {
				return nil, fmt.Errorf("Account '%v' not found", entryMap["account"])
			}
			result[i] = Entry{
				Account: db.Key{keys[0]},
				Value:   util.Round(entryMap["value"].(float64)*100) / 100}
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

	var transactionKey db.Key
	isUpdate := false
	if _, ok := param["transaction"]; ok {
		transaction.Key, err = db.DecodeKey(param["transaction"])
		if err != nil {
			return
		}
		isUpdate = true
	}

	transactionKey, err = db.Save(c, transaction, "Transaction", param["coa"], param)
	if err != nil {
		return
	}

	if isUpdate {
		if err = memcache.Delete(c, "transactions_asof_"+coaKey.Encode()); err != nil && err != memcache.ErrCacheMiss {
			return nil, err
		}
		if err = memcache.Delete(c, "balances_asof_"+coaKey.Encode()); err != nil && err != memcache.ErrCacheMiss {
			return nil, err
		}
		err = nil
	} else {
		cacheItem := &memcache.Item{
			Key:    "transactions_asof_" + coaKey.Encode(),
			Object: asOf,
		}
		if err = memcache.Gob.Set(c, cacheItem); err != nil {
			return nil, err
		}
	}

	if err = memcache.Delete(c, "transactions_"+coaKey.Encode()); err == memcache.ErrCacheMiss {
		err = nil
	}

	transaction.Key = transactionKey
	item = transaction

	return
}

func DeleteTransaction(c appengine.Context, m map[string]interface{}, param map[string]string, userKey core.UserKey) (_ interface{}, err error) {

	key, err := db.DecodeKey(param["transaction"])
	if err != nil {
		return
	}

	err = datastore.Delete(c, key.DsKey)
	if err != nil {
		return
	}

	if err = memcache.Delete(c, "transactions_asof_"+key.Parent().Encode()); err != nil && err != memcache.ErrCacheMiss {
		return nil, err
	}
	if err = memcache.Delete(c, "balances_asof_"+key.Parent().Encode()); err != nil && err != memcache.ErrCacheMiss {
		return nil, err
	}
	if err = memcache.Delete(c, "transactions_"+key.Parent().Encode()); err != nil && err != memcache.ErrCacheMiss {
		return nil, err
	}

	err = nil

	return

}

func Accounts(c appengine.Context, coaKey string, filters map[string]interface{}) (keys db.Keys, accounts []*Account, err error) {

	key, err := db.DecodeKey(coaKey)
	if err != nil {
		return
	}

	keys, items, err := db.GetAllFromCache(c, "Account", key, &accounts, []*Account{}, filters, []string{"Number"}, "accounts_"+coaKey)
	if err != nil {
		return
	}
	accounts = items.([]*Account)

	return
}

func Transactions(c appengine.Context, coaKey string, filters map[string]interface{}) (keys db.Keys, transactions []*Transaction, err error) {
	keys, items, err := db.GetAll(c, "Transaction", coaKey, &transactions, filters, []string{"Date", "AsOf"})
	if err != nil {
		return
	}
	return keys, *items.(*[]*Transaction), err
}

type TransactionWithValue struct {
	Transaction
	Value float64
}

func TransactionsWithValue(c appengine.Context, coaKey string, account *Account, from, to time.Time) (transactions []*TransactionWithValue, balance float64, err error) {

	key, err := db.DecodeKey(coaKey)
	if err != nil {
		return
	}

	q := datastore.NewQuery("Transaction").Ancestor(key.DsKey).Filter("AccountsKeysAsString =", account.Key.Encode()).Filter("Date >=", from).Filter("Date <=", to).Order("Date").Order("AsOf")
	keys, err := q.GetAll(c, &transactions)

	b, err := Balances(c, coaKey, time.Date(1000, 1, 1, 0, 0, 0, 0, time.UTC), from.AddDate(0, 0, -1), map[string]interface{}{"Number =": account.Number})
	if err != nil {
		return
	}
	balance = 0.0
	if len(b) > 0 {
		balance = b[0]["value"].(float64)
	}
	var (
		t *TransactionWithValue
		i int
	)
	lookupAccount := func(key db.Key) *Account {
		if key.String() == account.Key.String() {
			return account
		}
		return nil
	}
	addValue := func(key db.Key, value float64) {
		t.Value += value
	}
	for i, t = range transactions {
		t.Key = db.Key{keys[i]}
		t.incrementValue(lookupAccount, addValue)
	}
	return
}

type ByDateAndAsOf []*Transaction

func (a ByDateAndAsOf) Len() int      { return len(a) }
func (a ByDateAndAsOf) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByDateAndAsOf) Less(i, j int) bool {
	return a[i].Date.Before(a[j].Date) ||
		(a[i].Date.Equal(a[j].Date) && a[i].AsOf.Before(a[j].AsOf))
}

func Balances(c appengine.Context, coaKey string, from, to time.Time, accountFilters map[string]interface{}) (result []map[string]interface{}, err error) {

	key, err := db.DecodeKey(coaKey)
	if err != nil {
		return
	}

	var transactionsAsOf, balancesAsOf time.Time

	_, err = memcache.Gob.Get(c, "transactions_asof_"+coaKey, &transactionsAsOf)
	if err == memcache.ErrCacheMiss {
		q := datastore.NewQuery("Transaction").Ancestor(key.DsKey).Order("-AsOf").Limit(1)
		var transactions []*Transaction
		_, err = q.GetAll(c, &transactions)
		if err != nil {
			return nil, err
		}
		if len(transactions) == 1 {
			transactionsAsOf = transactions[0].AsOf
			cacheItem := &memcache.Item{
				Key:    "transactions_asof_" + coaKey,
				Object: transactionsAsOf,
			}
			if err = memcache.Gob.Set(c, cacheItem); err != nil {
				return nil, err
			}
		}
	} else if err != nil {
		return
	}

	timespanAsString := from.String() + "_" + to.String()

	var balancesAsOfMap map[string]time.Time
	_, err = memcache.Gob.Get(c, "balances_asof_"+coaKey, &balancesAsOfMap)
	if err != nil && err != memcache.ErrCacheMiss {
		return
	} else if err == nil {
		balancesAsOf = balancesAsOfMap[timespanAsString]
	}

	_, err = memcache.Gob.Get(c, "balances_"+coaKey+"_"+timespanAsString, &result)

	if err == memcache.ErrCacheMiss ||
		(err == nil && (transactionsAsOf != balancesAsOf || balancesAsOf.IsZero())) {

		accountKeys, accounts, err := Accounts(c, coaKey, nil)
		if err != nil {
			return nil, err
		}

		resultMap := map[string]map[string]interface{}{}

		q := datastore.NewQuery("Transaction").Ancestor(key.DsKey)

		if transactionsAsOf != balancesAsOf &&
			!transactionsAsOf.IsZero() && !balancesAsOf.IsZero() {
			q = q.Filter("AsOf >", balancesAsOf).Filter("AsOf <=", transactionsAsOf)
			for _, item := range result {
				resultMap[item["account"].(*Account).Key.String()] = item
			}
		} else {
			q = q.Filter("Date >=", from).Filter("Date <=", to)
			result = []map[string]interface{}{}
			for i, a := range accounts {
				a.Key = accountKeys.KeyAt(i)
				item := map[string]interface{}{"account": a, "value": 0.0}
				result = append(result, item)
				resultMap[accountKeys.KeyAt(i).String()] = item
			}
		}

		var transactions []*Transaction
		_, err = q.GetAll(c, &transactions)
		if err != nil {
			return nil, err
		}

		sort.Sort(ByDateAndAsOf(transactions))

		lookupAccount := func(key db.Key) *Account {
			if key.IsNil() {
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
			item["value"] = util.Round(item["value"].(float64)*100) / 100
		}
		for _, t := range transactions {
			if !t.Date.Before(from) && !t.Date.After(to) {
				t.incrementValue(lookupAccount, addValue)
			}
		}

		item := &memcache.Item{
			Key:    "balances_" + coaKey + "_" + timespanAsString,
			Object: result,
		}
		if err = memcache.Gob.Set(c, item); err != nil {
			return nil, err
		}
		if balancesAsOfMap == nil {
			balancesAsOfMap = map[string]time.Time{}
		}
		balancesAsOfMap[timespanAsString] = transactionsAsOf
		item = &memcache.Item{
			Key:    "balances_asof_" + coaKey,
			Object: balancesAsOfMap,
		}
		if err = memcache.Gob.Set(c, item); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	if accountFilters != nil {
		filteredResult := []map[string]interface{}{}
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
