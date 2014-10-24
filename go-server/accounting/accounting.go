package accounting

import (
	"fmt"
	"github.com/mcesarhm/geek-accounting/go-server/context"
	"github.com/mcesarhm/geek-accounting/go-server/core"
	"github.com/mcesarhm/geek-accounting/go-server/db"
	"github.com/mcesarhm/geek-accounting/go-server/util"
	_ "log"
	"sort"
	"strings"
	"time"
)

type ChartOfAccounts struct {
	db.Identifiable
	Name                    string       `json:"name"`
	RetainedEarningsAccount db.CKey      `json:"retainedEarningsAccount"`
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
	Number string       `json:"number"`
	Name   string       `json:"name"`
	Tags   []string     `json:"tags"`
	Parent db.CKey      `json:"parent"`
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

func (account *Account) ValidationMessage(db db.Db, param map[string]string) string {
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
	if account.Key.IsZero() {
		if key, err := accountKeyWithNumber(db, account.Number, param["coa"]); err != nil {
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
	Account db.CKey `json:"account"`
	Value   float64 `json:"value"`
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
	if util.Round(debitsSum*100) != util.Round(creditsSum*100) {
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

func (transaction *Transaction) incrementValue(lookupAccount func(db.Key) *Account, addValue func(db.Key, float64)) {
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

func AllChartsOfAccounts(c context.Context, _ map[string]string, _ core.UserKey) (interface{}, error) {
	_, chartsOfAccounts, err := c.Db.GetAll("ChartOfAccounts", "", &[]ChartOfAccounts{}, nil, []string{"Name"})
	return chartsOfAccounts, err
}

func SaveChartOfAccounts(c context.Context, m map[string]interface{}, param map[string]string, userKey core.UserKey) (interface{}, error) {
	coa := &ChartOfAccounts{
		Name: m["name"].(string),
		User: userKey,
		AsOf: time.Now()}
	_, err := c.Db.Save(coa, "ChartOfAccounts", "", param)
	return coa, err
}

func AllAccounts(c context.Context, param map[string]string, _ core.UserKey) (interface{}, error) {
	_, accounts, err := c.Db.GetAll("Account", param["coa"], &[]Account{}, nil, []string{"Number"})
	return accounts, err
}

func GetAccount(c context.Context, param map[string]string, _ core.UserKey) (result interface{}, err error) {
	return c.Db.Get(&Account{}, param["account"])
}

func SaveAccount(c context.Context, m map[string]interface{}, param map[string]string, userKey core.UserKey) (item interface{}, err error) {

	account := &Account{
		Number: m["number"].(string),
		Name:   m["name"].(string),
		Tags:   []string{},
		User:   userKey,
		AsOf:   time.Now()}

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
	}
	if parentNumber, ok := m["parent"]; ok {
		var accounts []Account
		keys, _, err := c.Db.GetAll("Account", param["coa"], &accounts, db.M{"Number = ": parentNumber}, nil)
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

	err = c.Db.Execute(func() (err error) {

		accountKey, err := c.Db.Save(account, "Account", param["coa"], param)
		if err != nil {
			return
		}

		if retainedEarningsAccount {
			coa := new(ChartOfAccounts)
			if _, err = c.Db.Get(coa, param["coa"]); err != nil {
				return
			}
			coa.RetainedEarningsAccount = accountKey.(db.CKey)
			if _, err = c.Db.Save(coa, "ChartOfAccounts", "", param); err != nil {
				return
			}
		}

		if !account.Parent.IsZero() && !isUpdate {
			i := util.IndexOf(parent.Tags, "analytic")
			if i != -1 {
				parent.Tags = append(parent.Tags[:i], parent.Tags[i+1:]...)
			}
			if !util.Contains(parent.Tags, "synthetic") {
				parent.Tags = append(parent.Tags, "synthetic")
			}
			if _, err = c.Db.Save(parent, "Account", param["coa"], param); err != nil {
				return
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

func DeleteAccount(c context.Context, m map[string]interface{}, param map[string]string, userKey core.UserKey) (_ interface{}, err error) {

	key, err := c.Db.DecodeKey(param["account"])
	if err != nil {
		return
	}

	coaKey, err := c.Db.DecodeKey(param["coa"])
	if err != nil {
		return
	}

	checkReferences := func(kind, property, errorMessage string) error {
		if keys, _, err := c.Db.GetAll(kind, param["coa"], nil, db.M{property: key}, nil); err != nil {
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
	err = checkReferences("Transaction", "Debits.Account = ", "Transactions referencing this account was found")
	if err != nil {
		return
	}
	err = checkReferences("Transaction", "Credits.Account = ", "Transactions referencing this account was found")
	if err != nil {
		return
	}

	if err = c.Db.Delete(key); err != nil {
		return
	}

	err = c.Cache.Delete("accounts_" + coaKey.Encode())

	return

}

func AllTransactions(c context.Context, param map[string]string, _ core.UserKey) (interface{}, error) {
	_, transactions, err := c.Db.GetAll("Transaction", param["coa"], &[]Transaction{}, nil, []string{"Date", "AsOf"})
	return transactions, err
}

func GetTransaction(c context.Context, param map[string]string, _ core.UserKey) (result interface{}, err error) {
	return c.Db.Get(&Transaction{}, param["transaction"])
}

func SaveTransaction(c context.Context, m map[string]interface{}, param map[string]string, userKey core.UserKey) (item interface{}, err error) {

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
			if key, err := accountKeyWithNumber(c.Db, entryMap["account"].(string), param["coa"]); err != nil {
				return nil, err
			} else if key.IsZero() {
				return nil, fmt.Errorf("Account '%v' not found", entryMap["account"])
			} else {
				result[i] = Entry{
					Account: key.(db.CKey),
					Value:   util.Round(entryMap["value"].(float64)*100) / 100}
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

	var transactionKey db.Key
	isUpdate := false
	if _, ok := param["transaction"]; ok {
		if k, err := c.Db.DecodeKey(param["transaction"]); err != nil {
			return nil, err
		} else {
			transaction.SetKey(k)
		}
		isUpdate = true
	}

	transaction.updateAccountsKeysAsString()

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
	item = transaction

	return
}

func DeleteTransaction(c context.Context, m map[string]interface{}, param map[string]string, userKey core.UserKey) (_ interface{}, err error) {

	key, err := c.Db.DecodeKey(param["transaction"])
	if err != nil {
		return
	}

	if err = c.Db.Delete(key); err != nil {
		return
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

	return

}

func Accounts(c context.Context, coaKey string, filters map[string]interface{}) (keys db.Keys, accounts []*Account, err error) {
	keys, _, err = c.Db.GetAllFromCache("Account", coaKey, &accounts, filters, []string{"Number"}, c.Cache, "accounts_"+coaKey)
	if err != nil {
		return
	}
	return
}

func Transactions(c context.Context, coaKey string, filters map[string]interface{}) (keys db.Keys, transactions []*Transaction, err error) {
	keys, _, err = c.Db.GetAll("Transaction", coaKey, &transactions, filters, []string{"Date", "AsOf"})
	if err != nil {
		return
	}
	return
}

type TransactionWithValue struct {
	Transaction
	Value float64
}

func TransactionsWithValue(c context.Context, coaKey string, account *Account, from, to time.Time) (transactionsWithValue []*TransactionWithValue, balance float64, err error) {

	var keys db.Keys
	var transactions []*Transaction
	if keys, _, err = c.Db.GetAll("Transaction", coaKey, &transactions, db.M{"AccountsKeysAsString =": account.Key.Encode(), "Date >=": from, "Date <=": to}, []string{"Date", "AsOf"}); err != nil {
		return
	}

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
	for i, t_ := range transactions {
		t = &TransactionWithValue{*t_, 0}
		t.SetKey(keys.KeyAt(i))
		t.incrementValue(lookupAccount, addValue)
		transactionsWithValue = append(transactionsWithValue, t)
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
			item["value"] = util.Round(item["value"].(float64)*100) / 100
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

func accountKeyWithNumber(d db.Db, number, coa string) (db.Key, error) {
	if keys, _, err := d.GetAll("Account", coa, nil, db.M{"Number = ": number}, nil); err != nil {
		return d.NewKey(), err
	} else if keys.Len() == 0 {
		return d.NewKey(), nil
	} else {
		return keys.KeyAt(0), nil
	}
}
