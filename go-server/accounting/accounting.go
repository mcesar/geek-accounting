package accounting

import (
	//"log"
	"fmt"
	"reflect"
	"strings"
	"time"
	"appengine"
	"appengine/datastore"
)

type ChartOfAccounts struct {
	Key *datastore.Key `datastore:"-" json:"_id"`
	Name string `json:"name"`
	RetainedEarningsAccount *datastore.Key `json:"retainedEarningsAccount"`
	User *datastore.Key `json:"user"`
	AsOf time.Time `json:"timestamp"`
}

func (coa *ChartOfAccounts) ValidationMessage(_ appengine.Context, _ map[string]string) string {
	if len(strings.TrimSpace(coa.Name)) == 0 {
		return "The name must be informed"
	}
	return ""
}

type Account struct {
	Key *datastore.Key `datastore:"-" json:"_id"`
	Number string `json:"number"`
	Name string `json:"name"`
	Tags []string `json:"tags"`
	Parent *datastore.Key `json:"parent"`
	User *datastore.Key `json:"user"`
	AsOf time.Time `json:"timestamp"`
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
	coaKey, err := datastore.DecodeKey(param["coa"])
	if err != nil {
		return err.Error()
	}
	if account.Key == nil {
		q := datastore.NewQuery("Account").Ancestor(coaKey).Filter("Number = ", account.Number).KeysOnly()
		keys, err := q.GetAll(c, nil)
		if err != nil {
			return err.Error()
		}
		if len(keys) != 0 {
			return "An account with this number already exists"
		}		
	}
	if account.Parent != nil {
		var parent Account
		if err := datastore.Get(c, account.Parent, &parent); err != nil {
			return err.Error()
		}
		if !strings.HasPrefix(account.Number, parent.Number) {
			return "The number must start with parent's number"
		}
		for key, value := range inheritedProperties {
			if contains(parent.Tags, key) && !contains(account.Tags, key) {
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
	if contains(account.Tags, "debitBalance") {
		return value
	} else {
		return -value
	}
}

func (account *Account) Credit(value float64) float64 {
	if contains(account.Tags, "creditBalance") {
		return value
	} else {
		return -value
	}
}

type Transaction struct {
	Key *datastore.Key `datastore:"-" json:"_id"`
	Debits []Entry `json:"debits"`
	Credits []Entry `json:"credits"`
	Date time.Time `json:"date`
	Memo string `json:"memo"`
	Tags []string `json:"tags"`
	User *datastore.Key `json:"user"`
	AsOf time.Time `json:"timestamp"`
}

type Entry struct {
	Account *datastore.Key `json:"account"`
	Value float64 `json:"value"`
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
	if debitsSum != creditsSum {
		return "The sum of debit values must be equals to the sum of credit values"
	}
	return ""
}

func (entry *Entry) ValidationMessage(c appengine.Context, param map[string]string) string {
	if entry.Account == nil {
		return "The account must be informed for each entry"
	}
	var account = new(Account)
	if err := datastore.Get(c, entry.Account, account); err != nil {
		return err.Error()
	}
	if account == nil {
		return "Account not found"
	}
	if !contains(account.Tags, "analytic") {
		return "The account must be analytic"
	}
	coaKey, err := datastore.DecodeKey(param["coa"])
	if err != nil {
		return err.Error()
	}
	if entry.Account.Parent().String() != coaKey.String() {
		return "The account must belong to the same chart of accounts of the transaction"
	}

	return ""
}

func AllChartsOfAccounts(c appengine.Context, _ map[string]string, _ *datastore.Key) (interface{}, error) {
	return getAll(c, &[]ChartOfAccounts{}, "ChartOfAccounts", "", []string{"Name"})
}

func SaveChartOfAccounts(c appengine.Context, m map[string]interface{}, param map[string]string, userKey *datastore.Key) (interface{}, error) {
	coa := &ChartOfAccounts{
		Name: m["name"].(string), 
		User: userKey,
		AsOf: time.Now()}
	_, err := save(c, coa, "ChartOfAccounts", "", param)
	return coa, err
}

func AllAccounts(c appengine.Context, param map[string]string, _ *datastore.Key) (interface{}, error) {
	return getAll(c, &[]Account{}, "Account", param["coa"], []string{"Number"})
}

func SaveAccount(c appengine.Context, m map[string]interface{}, param map[string]string, userKey *datastore.Key) (item interface{}, err error) {

	account := &Account{
		Number: m["number"].(string), 
		Name: m["name"].(string), 
		Tags: []string{},
		User: userKey,
		AsOf: time.Now()}

	if accountKeyAsString, ok := param["account"]; ok {
		account.Key, err = datastore.DecodeKey(accountKeyAsString)
		if err != nil {
			return
		}
	}

	coaKey, err := datastore.DecodeKey(param["coa"])
	if err != nil {
		return
	}

	var parent []Account
	if parentNumber, ok := m["parent"]; ok {
		q := datastore.NewQuery("Account").Ancestor(coaKey).Filter("Number = ", parentNumber)
		keys, err := q.GetAll(c, &parent)
		if err != nil {
			return nil, err
		}
		if len(keys) == 0 {
			return nil, fmt.Errorf("Parent not found")
		}
		account.Parent = keys[0]
		delete(m, "parent")
	}

	var retainedEarningsAccount bool
	for k, _ := range m {
		if k != "name" && k != "number" {
			if k == "retainedEarnings" {
				retainedEarningsAccount = true
			} else {
				account.Tags = append(account.Tags, k)
			}
		}
	}
	if !contains(account.Tags, "analytic") {
		account.Tags = append(account.Tags, "analytic")
	}

	err = datastore.RunInTransaction(c, func(c appengine.Context) (err error) {

		accountKey, err := save(c, account, "Account", param["coa"], param)
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

		if account.Parent != nil {
			i := indexOf(parent[0].Tags, "analytic")
			if i != -1 {
				parent[0].Tags = append(parent[0].Tags[:i], parent[0].Tags[i+1:]...)
			}
			if !contains(parent[0].Tags, "synthetic") {
				parent[0].Tags = append(parent[0].Tags, "synthetic")
			}
			if _, err = datastore.Put(c, account.Parent, &parent[0]); err != nil {
				return
			}
		}
		return
	}, nil)
	if err != nil {
		return
	}

	item = account
	return
}

func AllTransactions(c appengine.Context, param map[string]string, _ *datastore.Key) (interface{}, error) {
	return getAll(c, &[]Transaction{}, "Transaction", param["coa"], []string{"Date", "AsOf"})
}

func SaveTransaction(c appengine.Context, m map[string]interface{}, param map[string]string, userKey *datastore.Key) (item interface{}, err error) {

	transaction := &Transaction{
		Memo: m["memo"].(string),
		AsOf: time.Now(),
		User: userKey}
	transaction.Date, err = time.Parse(time.RFC3339, m["date"].(string))
	if err != nil { return }

	coaKey, err := datastore.DecodeKey(param["coa"])
	if err != nil { return }

	entries := func(property string) (result []Entry, err error) {
		for _, each := range m[property].([]interface{}) {

			entry := each.(map[string]interface{})

			q := datastore.NewQuery("Account").Ancestor(coaKey).Filter("Number = ", entry["account"]).KeysOnly()
			var keys []*datastore.Key
			if keys, err = q.GetAll(c, nil); err != nil {
				return
			}
			if len(keys) == 0 {
				return nil, fmt.Errorf("Account '%v' not found", entry["account"])
			}

			result = append(result, Entry{Account: keys[0], Value: entry["value"].(float64)})
		}
		return
	}
	if transaction.Debits, err = entries("debits"); err != nil {
		return
	}
	if transaction.Credits, err = entries("credits"); err != nil {
		return
	}

	transactionKey, err := save(c, transaction, "Transaction", param["coa"], param)
	if err != nil {
		return
	}
	transaction.Key = transactionKey
	item = transaction

	return
}

func Balance(c appengine.Context, m map[string]string, _ *datastore.Key) (result interface{}, err error) {
	coaKey, err := datastore.DecodeKey(m["coa"])
	if err != nil { return }
	from := time.Date(1000, 1, 1, 0, 0, 0, 0, time.UTC)
	to, err := time.Parse(time.RFC3339, m["at"] + "T00:00:00Z")
	if err != nil { return }
	return balances(c, coaKey, from, to, map[string]interface{}{"Tags =": "balanceSheet"})
}

func Journal(c appengine.Context, m map[string]string, _ *datastore.Key) (result interface{}, err error) {

	coaKey, err := datastore.DecodeKey(m["coa"])
	if err != nil { return }
	from, err := time.Parse(time.RFC3339, m["from"] + "T00:00:00Z")
	to, err := time.Parse(time.RFC3339, m["to"] + "T00:00:00Z")
	if err != nil { return }

	q := datastore.NewQuery("Account").Ancestor(coaKey).Order("Number")
	var accounts []*Account
	accountKeys, err := q.GetAll(c, &accounts)
	if err != nil { return }

	q = datastore.NewQuery("Transaction").Ancestor(coaKey).Filter("Date >=", from).Filter("Date <=", to).Order("Date").Order("AsOf")
	var transactions []*Transaction
	transactionKeys, err := q.GetAll(c, &transactions)
	if err != nil { return }

	accountsMap := map[string]*Account{}
	for i, a := range accounts {
		accountsMap[accountKeys[i].String()] = a
	}

	resultMap := []map[string]interface{}{}

	addEntries := func(entries []Entry) (result []map[string]interface{}) {
		for _, e := range entries {
			account := accountsMap[e.Account.String()]
			result = append(result, map[string]interface{}{
				"account": map[string]interface{}{
					"number": account.Number,
					"name": account.Name,
				},
				"value": e.Value,
			})
		}
		return
	}

	for i, t := range transactions {
		m := map[string]interface{}{
			"_id": transactionKeys[i],
			"date": t.Date,
			"memo": t.Memo,
			"debits": addEntries(t.Debits),
			"credits": addEntries(t.Credits),
		}
		resultMap = append(resultMap, m)
	}

	result = resultMap

	return
}

func Ledger(c appengine.Context, m map[string]string, _ *datastore.Key) (result interface{}, err error) {

	coaKey, err := datastore.DecodeKey(m["coa"])
	if err != nil { return }
	from, err := time.Parse(time.RFC3339, m["from"] + "T00:00:00Z")
	to, err := time.Parse(time.RFC3339, m["to"] + "T00:00:00Z")
	if err != nil { return }
	var (
		accountKey *datastore.Key
		account *Account
	)
	q := datastore.NewQuery("Account").Ancestor(coaKey)
	var accounts []*Account
	accountKeys, err := q.GetAll(c, &accounts)
	if err != nil { return }

	accountsMap := map[string]*Account{}
	for i, a := range accounts {
		if a.Number == m["account"] {
			accountKey = accountKeys[i]
			account = a
		}
		accountsMap[accountKeys[i].String()] = a
	}	

	q = datastore.NewQuery("Transaction").Ancestor(coaKey).Filter("Date >=", from).Filter("Date <=", to).Order("Date").Order("AsOf")
	var transactions []*Transaction
	_, err = q.GetAll(c, &transactions)
	if err != nil { return }

	resultEntries := []interface{}{}
	runningBalance, balance := 0.0, 0.0

	addEntries := func(t *Transaction, entries []Entry, counterpartEntries []Entry, f func(*Account, float64) float64, kind string) {
		for _, e := range entries {
			if e.Account.String() != accountKey.String() { continue }
			runningBalance += f(accountsMap[e.Account.String()], e.Value)
			if t.Date.Before(from) {
				balance = runningBalance
			} else {
				entry := map[string]interface{}{
					"date": t.Date,
					"memo": t.Memo,
					"balance": runningBalance,
				}
				entry[kind] = e.Value
				counterpart := map[string]interface{}{}
				entry["counterpart"] = counterpart
				if len(counterpartEntries) == 1 {
					counterpartAccount := accountsMap[counterpartEntries[0].Account.String()]
					counterpart["number"] = counterpartAccount.Number
					counterpart["name"] = counterpartAccount.Name
				} else {
					counterpart["name"] = "many"
				}
				resultEntries = append(resultEntries, entry)
			}
		}
		return
	}

	for _, t := range transactions {
		addEntries(t, t.Debits, t.Credits, (*Account).Debit, "debit")
		addEntries(t, t.Credits, t.Debits, (*Account).Credit, "credit")
	}

	result = map[string]interface{}{
		"account": map[string]interface{}{
			"_id": accountKey,
			"number": account.Number,
			"name": account.Name,
			"debitBalance": contains(account.Tags, "debitBalance"),
			"creditBalance": contains(account.Tags, "creditBalance"),
		},
		"entries": resultEntries,
		"balance": balance,
	} 
	
	return
}

func IncomeStatement(c appengine.Context, m map[string]string, _ *datastore.Key) (result interface{}, err error) {
	coaKey, err := datastore.DecodeKey(m["coa"])
	if err != nil { return }
	from, err := time.Parse(time.RFC3339, m["from"] + "T00:00:00Z")
	if err != nil { return }
	to, err := time.Parse(time.RFC3339, m["to"] + "T00:00:00Z")
	if err != nil { return }

	balances, err := balances(c, coaKey, from, to, map[string]interface{}{"Tags =": "incomeStatement"})
	if err != nil { return }

	type entryType struct {
		Balance float64 `json:"balance"`
		Details[]interface{} `json:"details"`
	}
	type resultType struct {
		GrossRevenue *entryType `json:"grossRevenue"`
		Deduction *entryType `json:"deduction"`
		SalesTax *entryType `json:"salesTax"`
		NetRevenue *entryType `json:"netRevenue"`
		Cost *entryType `json:"cost"`
		GrossProfit *entryType `json:"grossProfit"`
		OperatingExpense *entryType `json:"operatingExpense"`
		NetOperatingIncome *entryType `json:"netOperatingIncome"`
		NonOperatingRevenue *entryType `json:"nonOperatingRevenue"`
		NonOperatingExpense *entryType `json:"nonOperatingExpense"`
		NonOperatingTax *entryType `json:"nonOperatingTax"`
		IncomeBeforeIncomeTax *entryType `json:"incomeBeforeIncomeTax"`
		IncomeTax *entryType `json:"incomeTax"`
		Dividends *entryType `json:"dividends"`
		NetIncome *entryType `json:"netIncome"`
	}

	var (
		resultTyped resultType
		revenueRoots, expenseRoots []*Account
	)

	addBalance := func(entry *entryType, balance map[string]interface{}) *entryType {
		if contains(balance["account"].(*Account).Tags, "analytic") && balance["value"].(float64) > 0 {
			if entry == nil {
				entry = &entryType{}
			}
			entry.Balance += balance["value"].(float64)
			entry.Details = append(entry.Details, balance)
		}
		return entry
	}

	isDescendent := func(account *Account, parents []*Account) bool {
		for _, p := range parents {
			if strings.HasPrefix(account.Number, p.Number) {
				return true
			}
		}
		return false
	}

	collectRoots := func(parent *datastore.Key) {
		for _, m := range balances {
			account := m["account"].(*Account)
			if (account.Parent == nil && parent == nil) || account.Parent.String() == parent.String() {
				if contains(account.Tags, "creditBalance") {
					revenueRoots = append(revenueRoots, account)
				} else {
					expenseRoots = append(expenseRoots, account)
				}
			}
		}		
	}

	collectRoots(nil)

	if (len(revenueRoots) + len(expenseRoots)) == 1 {
		collectRoots(append(revenueRoots, expenseRoots...)[0].Key)
	}

	for _, m := range balances {
		account := m["account"].(*Account)
		if contains(account.Tags, "operating") && isDescendent(account, revenueRoots) {
			resultTyped.GrossRevenue = addBalance(resultTyped.GrossRevenue, m)
		} else if contains(account.Tags, "deduction") {
			resultTyped.Deduction = addBalance(resultTyped.Deduction, m)
		} else if contains(account.Tags, "salesTax") {
			resultTyped.SalesTax = addBalance(resultTyped.SalesTax, m)
		} else if contains(account.Tags, "cost") {
			resultTyped.Cost = addBalance(resultTyped.Cost, m)
		} else if contains(account.Tags, "operating") && isDescendent(account, expenseRoots) {
			resultTyped.OperatingExpense = addBalance(resultTyped.OperatingExpense, m)
		} else if contains(account.Tags, "nonOperatingTax") {
			resultTyped.NonOperatingTax = addBalance(resultTyped.NonOperatingTax, m)
		} else if contains(account.Tags, "incomeTax") {
			resultTyped.IncomeTax = addBalance(resultTyped.IncomeTax, m)
		} else if contains(account.Tags, "dividends") {
			resultTyped.Dividends = addBalance(resultTyped.Dividends, m)
		} else if isDescendent(account, revenueRoots) {
			resultTyped.NonOperatingRevenue = addBalance(resultTyped.NonOperatingRevenue, m)
		} else {
			resultTyped.NonOperatingExpense = addBalance(resultTyped.NonOperatingExpense, m)
		}
	}

	ze := &entryType{}
	z := func(e *entryType) *entryType {
		if e == nil {
			return ze
		} else {
			return e
		}
	}

	resultTyped.NetRevenue = &entryType{
		Balance: z(resultTyped.GrossRevenue).Balance - z(resultTyped.Deduction).Balance - 
		z(resultTyped.SalesTax).Balance}
	resultTyped.GrossProfit = &entryType{
		Balance: z(resultTyped.NetRevenue).Balance - 
		z(resultTyped.Cost).Balance}
	resultTyped.NetOperatingIncome = &entryType{
		Balance: z(resultTyped.GrossProfit).Balance - 
		z(resultTyped.OperatingExpense).Balance}
	resultTyped.IncomeBeforeIncomeTax = &entryType{
		Balance: z(resultTyped.NetOperatingIncome).Balance + 
		z(resultTyped.NonOperatingRevenue).Balance -
		z(resultTyped.NonOperatingExpense).Balance - z(resultTyped.NonOperatingTax).Balance}
	resultTyped.NetIncome = &entryType{
		Balance: z(resultTyped.IncomeBeforeIncomeTax).Balance - 
		z(resultTyped.IncomeTax).Balance - z(resultTyped.Dividends).Balance}

	if resultTyped.NetRevenue.Balance == 0 { resultTyped.NetRevenue = nil }
	if resultTyped.GrossProfit.Balance == 0 { resultTyped.GrossProfit = nil }
	if resultTyped.NetOperatingIncome.Balance == 0 { resultTyped.NetOperatingIncome = nil }
	if resultTyped.IncomeBeforeIncomeTax.Balance == 0 { resultTyped.IncomeBeforeIncomeTax = nil }

	result = resultTyped

	return 
}

func balances(c appengine.Context, coaKey *datastore.Key, from, to time.Time, accountFilters map[string]interface{}) (result []map[string]interface{}, err error) {
	
	q := datastore.NewQuery("Account").Ancestor(coaKey).Order("Number")
	for k, v := range accountFilters {
		q = q.Filter(k, v)
	}
	var accounts []*Account
	accountKeys, err := q.GetAll(c, &accounts)
	if err != nil { return }

	q = datastore.NewQuery("Transaction").Ancestor(coaKey).Filter("Date >=", from).Filter("Date <=", to).Order("Date").Order("AsOf")
	var transactions []*Transaction
	_, err = q.GetAll(c, &transactions)
	if err != nil { return }

	resultMap := map[string]map[string]interface{}{}
	for i, a := range accounts {
		a.Key = accountKeys[i]
		item := map[string]interface{}{"account": a, "value": 0.0}
		result = append(result, item)
		resultMap[accountKeys[i].String()] = item
	}

	incrementValue := func(entries []Entry, f func(*Account, float64) float64) {
		for _, e := range entries {
			accountKey, value := e.Account, e.Value
			for accountKey != nil {
				item := resultMap[accountKey.String()]
				if item["account"] != nil {
					account := item["account"].(*Account)
					item["value"] = item["value"].(float64) + f(account, value)
					accountKey = account.Parent
				} else {
					accountKey = nil
				}
			}
		}
	}

	for _, t := range transactions {
		incrementValue(t.Debits, (*Account).Debit)
		incrementValue(t.Credits, (*Account).Credit)
	}

	return
}

func getAll(c appengine.Context, items interface{}, kind string, ancestor string, orderKeys []string) (interface{}, error) {
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