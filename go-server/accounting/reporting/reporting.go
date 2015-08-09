package reporting

import (
	"fmt"

	"github.com/mcesarhm/geek-accounting/go-server/accounting"
	"github.com/mcesarhm/geek-accounting/go-server/context"
	"github.com/mcesarhm/geek-accounting/go-server/core"
	"github.com/mcesarhm/geek-accounting/go-server/db"
	"github.com/mcesarhm/geek-accounting/go-server/extensions/collections"
	"mcesar.io/deb"
	//"log"
	"math"
	"sort"
	"strings"
	"time"
)

type byNumber []db.M

func (arr byNumber) Len() int      { return len(arr) }
func (arr byNumber) Swap(i, j int) { arr[i], arr[j] = arr[j], arr[i] }
func (arr byNumber) Less(i, j int) bool {
	a1 := arr[i]["account"].(map[string]interface{})
	a2 := arr[j]["account"].(map[string]interface{})
	return a1["number"].(string) < a2["number"].(string)
}

func Balance(c context.Context, m map[string]interface{}, param map[string]string,
	_ core.UserKey) (result interface{}, err error) {
	from := time.Date(1000, 1, 1, 0, 0, 0, 0, time.UTC)
	to, err := time.Parse(time.RFC3339, param["at"]+"T00:00:00Z")
	if err != nil {
		return
	}
	arr := []db.M{}
	space, ok := m["space"].(deb.Space)
	if !ok {
		b, err := accounting.Balances(c, param["coa"], from, to,
			map[string]interface{}{"Tags =": "balanceSheet"})
		if err != nil {
			return nil, err
		}
		for _, e := range b {
			account := e["account"].(*accounting.Account)
			arr = append(arr, db.M{
				"account": accountToMap(account.Key, account),
				"value":   e["value"]})
		}
	} else {
		accountKeys, accounts, err := accounting.Accounts(c, param["coa"], nil)
		if err != nil {
			return nil, err
		}
		sortedAccounts, sortedKeys := accounting.AccountsByCreation(accounts, accountKeys)
		balanceSpace, err := space.Projection(nil,
			[]deb.DateRange{deb.DateRange{
				Start: accounting.SerializedDate(from),
				End:   accounting.SerializedDate(to)}}, nil)
		if err != nil {
			return nil, err
		}
		ch, errc := balanceSpace.Transactions()
		for t := range ch {
			for k, v := range t.Entries {
				account := sortedAccounts[k-1]
				if collections.Contains(account.Tags, "balanceSheet") {
					value := float64(v) / 100
					if collections.Contains(account.Tags, "creditBalance") {
						value = -value
					}
					arr = append(arr,
						db.M{"account": accountToMap(sortedKeys[k-1], account), "value": value})
				}
			}
		}
		if err = <-errc; err != nil {
			return nil, err
		}
		sort.Sort(byNumber(arr))
	}
	result = arr
	return
}

func Journal(c context.Context, m map[string]interface{}, param map[string]string,
	_ core.UserKey) (result interface{}, err error) {

	from, err := time.Parse(time.RFC3339, param["from"]+"T00:00:00Z")
	if err != nil {
		return
	}
	to, err := time.Parse(time.RFC3339, param["to"]+"T00:00:00Z")
	if err != nil {
		return
	}

	accountKeys, accounts, err := accounting.Accounts(c, param["coa"], nil)
	if err != nil {
		return
	}

	space, ok := m["space"].(deb.Space)

	var transactions []*accounting.Transaction
	var transactionKeys []interface{}
	if !ok {
		var keys db.Keys
		keys, transactions, err = accounting.Transactions(c, param["coa"],
			map[string]interface{}{"Date >=": from, "Date <=": to})
		if err != nil {
			return nil, err
		}
		transactionKeys = make([]interface{}, len(keys))
		for i, k := range keys {
			transactionKeys[i] = k
		}
	} else {
		journal, err := space.Slice(nil,
			[]deb.DateRange{deb.DateRange{Start: accounting.SerializedDate(from),
				End: accounting.SerializedDate(to)}}, nil)
		if err != nil {
			return nil, err
		}
		transactions, transactionKeys, err =
			accounting.TransactionsFromSpace(journal, accounts, accountKeys)
		if err != nil {
			return nil, err
		}
	}

	accountsMap := map[string]*accounting.Account{}
	for i, a := range accounts {
		accountsMap[accountKeys.KeyAt(i).String()] = a
	}

	resultMap := []map[string]interface{}{}

	addEntries := func(entries []accounting.Entry) (result []map[string]interface{}) {
		for _, e := range entries {
			account := accountsMap[e.Account.String()]
			result = append(result, map[string]interface{}{
				"account": map[string]interface{}{
					"number": account.Number,
					"name":   account.Name,
				},
				"value": e.Value,
			})
		}
		return
	}

	for i, t := range transactions {
		m := map[string]interface{}{
			"_id":     transactionKeys[i],
			"date":    t.Date,
			"memo":    t.Memo,
			"debits":  addEntries(t.Debits),
			"credits": addEntries(t.Credits),
		}
		resultMap = append(resultMap, m)
	}

	result = resultMap

	return
}

func Ledger(c context.Context, m map[string]interface{}, param map[string]string,
	_ core.UserKey) (result interface{}, err error) {

	from, err := time.Parse(time.RFC3339, param["from"]+"T00:00:00Z")
	if err != nil {
		return
	}
	to, err := time.Parse(time.RFC3339, param["to"]+"T00:00:00Z")
	if err != nil {
		return
	}

	accountKeys, accounts, err := accounting.Accounts(c, param["coa"], nil)
	if err != nil {
		return
	}

	var account *accounting.Account
	accountsMap := map[string]*accounting.Account{}
	for i, a := range accounts {
		if a.Number == param["account"] || accountKeys.KeyAt(i).Encode() == param["account"] {
			account = a
			account.SetKey(accountKeys.KeyAt(i))
		}
		accountsMap[accountKeys.KeyAt(i).String()] = a
	}
	if account == nil {
		err = fmt.Errorf("Account not found")
		return
	}

	var transactions []*accounting.TransactionWithValue
	var balance float64
	space, ok := m["space"].(deb.Space)
	if !ok {
		transactions, balance, err =
			accounting.TransactionsWithValue(c, param["coa"], account, from, to)
		if err != nil {
			return nil, err
		}
	} else {
		_, sortedKeys := accounting.AccountsByCreation(accounts, accountKeys)
		var accountIndex deb.Account
		for i, k := range sortedKeys {
			if k == account.GetKey() {
				accountIndex = deb.Account(i + 1)
				break
			}
		}
		balanceSpace, err := space.Projection([]deb.Account{accountIndex},
			[]deb.DateRange{deb.DateRange{
				Start: deb.Date(0),
				End:   accounting.SerializedDate(from.AddDate(0, 0, -1))}}, nil)
		if err != nil {
			return nil, err
		}
		ch, errc := balanceSpace.Transactions()
		for t := range ch {
			balance = float64(t.Entries[accountIndex]) / 100
		}
		if err = <-errc; err != nil {
			return nil, err
		}
		if collections.Contains(account.Tags, "creditBalance") {
			balance = -balance
		}
		ledger, err := space.Slice([]deb.Account{accountIndex},
			[]deb.DateRange{deb.DateRange{
				Start: accounting.SerializedDate(from),
				End:   accounting.SerializedDate(to)}}, nil)
		if err != nil {
			return nil, err
		}
		txs, transactionKeys, err :=
			accounting.TransactionsFromSpace(ledger, accounts, accountKeys)
		if err != nil {
			return nil, err
		}
		transactions = accounting.TransactionsWithValueFromTransactions(txs, transactionKeys,
			account)
	}

	resultEntries := []interface{}{}
	runningBalance := balance
	addEntries := func(t *accounting.TransactionWithValue, entries []accounting.Entry, counterpartEntries []accounting.Entry, kind string) {
		found := false
		for _, e := range entries {
			if e.Account.String() == account.Key.String() {
				found = true
			}
		}
		if !found {
			return
		}
		runningBalance += t.Value
		entryMap := map[string]interface{}{
			"_id":     t.Key,
			"date":    t.Date,
			"memo":    t.Memo,
			"balance": runningBalance,
		}
		entryMap[kind] = math.Abs(t.Value)
		counterpart := map[string]interface{}{}
		entryMap["counterpart"] = counterpart
		if len(counterpartEntries) == 1 {
			counterpartAccount := accountsMap[counterpartEntries[0].Account.String()]
			counterpart["number"] = counterpartAccount.Number
			counterpart["name"] = counterpartAccount.Name
		} else {
			counterpart["name"] = "many"
		}
		resultEntries = append(resultEntries, entryMap)
		return
	}

	for _, t := range transactions {
		addEntries(t, t.Debits, t.Credits, "debit")
		addEntries(t, t.Credits, t.Debits, "credit")
	}

	result = map[string]interface{}{
		"account": accountToMap(account.Key, account),
		"entries": resultEntries,
		"balance": balance,
	}

	return
}

func IncomeStatement(c context.Context, m map[string]interface{}, param map[string]string,
	_ core.UserKey) (result interface{}, err error) {
	from, err := time.Parse(time.RFC3339, param["from"]+"T00:00:00Z")
	if err != nil {
		return
	}
	to, err := time.Parse(time.RFC3339, param["to"]+"T00:00:00Z")
	if err != nil {
		return
	}

	space, ok := m["space"].(deb.Space)
	var balances []db.M
	if !ok {
		balances, err = accounting.Balances(c, param["coa"], from, to,
			map[string]interface{}{"Tags =": "incomeStatement"})
		if err != nil {
			return nil, err
		}
	} else {
		accountKeys, accounts, err := accounting.Accounts(c, param["coa"], nil)
		if err != nil {
			return nil, err
		}
		sortedAccounts, sortedKeys := accounting.AccountsByCreation(accounts, accountKeys)
		balanceSpace, err := space.Projection(nil,
			[]deb.DateRange{deb.DateRange{
				Start: accounting.SerializedDate(from),
				End:   accounting.SerializedDate(to)}}, nil)
		if err != nil {
			return nil, err
		}
		ch, errc := balanceSpace.Transactions()
		for t := range ch {
			for k, v := range t.Entries {
				account := sortedAccounts[k-1]
				account.Key = sortedKeys[k-1]
				if collections.Contains(account.Tags, "incomeStatement") {
					value := float64(v) / 100
					if collections.Contains(account.Tags, "creditBalance") {
						value = -value
					}
					balances = append(balances, db.M{"account": account, "value": value})
				}
			}
		}
		if err = <-errc; err != nil {
			return nil, err
		}

	}

	type entryType struct {
		Balance float64       `json:"balance"`
		Details []interface{} `json:"details"`
	}
	type resultType struct {
		GrossRevenue          *entryType `json:"grossRevenue"`
		Deduction             *entryType `json:"deduction"`
		SalesTax              *entryType `json:"salesTax"`
		NetRevenue            *entryType `json:"netRevenue"`
		Cost                  *entryType `json:"cost"`
		GrossProfit           *entryType `json:"grossProfit"`
		OperatingExpense      *entryType `json:"operatingExpense"`
		NetOperatingIncome    *entryType `json:"netOperatingIncome"`
		NonOperatingRevenue   *entryType `json:"nonOperatingRevenue"`
		NonOperatingExpense   *entryType `json:"nonOperatingExpense"`
		NonOperatingTax       *entryType `json:"nonOperatingTax"`
		IncomeBeforeIncomeTax *entryType `json:"incomeBeforeIncomeTax"`
		IncomeTax             *entryType `json:"incomeTax"`
		Dividends             *entryType `json:"dividends"`
		NetIncome             *entryType `json:"netIncome"`
	}

	var (
		resultTyped                resultType
		revenueRoots, expenseRoots []*accounting.Account
	)

	addBalance := func(entry *entryType, balance map[string]interface{}) *entryType {
		if collections.Contains(balance["account"].(*accounting.Account).Tags, "analytic") &&
			balance["value"].(float64) > 0 {
			if entry == nil {
				entry = &entryType{}
			}
			entry.Balance += balance["value"].(float64)
			entry.Details = append(entry.Details, balance)
		}
		return entry
	}

	isDescendent := func(account *accounting.Account, parents []*accounting.Account) bool {
		for _, p := range parents {
			if strings.HasPrefix(account.Number, p.Number) {
				return true
			}
		}
		return false
	}

	collectRoots := func(parent db.Key) {
		for _, m := range balances {
			account := m["account"].(*accounting.Account)
			if (account.Parent.IsZero() && parent.IsZero()) ||
				account.Parent.String() == parent.String() {
				if collections.Contains(account.Tags, "creditBalance") {
					revenueRoots = append(revenueRoots, account)
				} else {
					expenseRoots = append(expenseRoots, account)
				}
			}
		}
	}

	collectRoots(c.Db.NewKey())

	if (len(revenueRoots) + len(expenseRoots)) == 1 {
		parentKey := append(revenueRoots, expenseRoots...)[0].Key
		revenueRoots = revenueRoots[0:0]
		expenseRoots = expenseRoots[0:0]
		collectRoots(parentKey)
	}

	for _, m := range balances {
		account := m["account"].(*accounting.Account)
		if collections.Contains(account.Tags, "operating") && isDescendent(account, revenueRoots) {
			resultTyped.GrossRevenue = addBalance(resultTyped.GrossRevenue, m)
		} else if collections.Contains(account.Tags, "deduction") {
			resultTyped.Deduction = addBalance(resultTyped.Deduction, m)
		} else if collections.Contains(account.Tags, "salesTax") {
			resultTyped.SalesTax = addBalance(resultTyped.SalesTax, m)
		} else if collections.Contains(account.Tags, "cost") {
			resultTyped.Cost = addBalance(resultTyped.Cost, m)
		} else if collections.Contains(account.Tags, "operating") &&
			isDescendent(account, expenseRoots) {
			resultTyped.OperatingExpense = addBalance(resultTyped.OperatingExpense, m)
		} else if collections.Contains(account.Tags, "nonOperatingTax") {
			resultTyped.NonOperatingTax = addBalance(resultTyped.NonOperatingTax, m)
		} else if collections.Contains(account.Tags, "incomeTax") {
			resultTyped.IncomeTax = addBalance(resultTyped.IncomeTax, m)
		} else if collections.Contains(account.Tags, "dividends") {
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

	if resultTyped.NetRevenue.Balance == 0 || (z(resultTyped.Deduction).Balance == 0 &&
		z(resultTyped.SalesTax).Balance == 0) {
		resultTyped.NetRevenue = nil
	}
	if resultTyped.GrossProfit.Balance == 0 || z(resultTyped.Cost).Balance == 0 {
		resultTyped.GrossProfit = nil
	}
	if resultTyped.NetOperatingIncome.Balance == 0 {
		resultTyped.NetOperatingIncome = nil
	}
	if resultTyped.IncomeBeforeIncomeTax.Balance == 0 ||
		z(resultTyped.NonOperatingTax).Balance == 0 {
		resultTyped.IncomeBeforeIncomeTax = nil
	}

	result = resultTyped

	return
}

func accountToMap(key interface{}, account *accounting.Account) map[string]interface{} {
	return map[string]interface{}{
		"_id":           key,
		"number":        account.Number,
		"name":          account.Name,
		"debitBalance":  collections.Contains(account.Tags, "debitBalance"),
		"creditBalance": collections.Contains(account.Tags, "creditBalance"),
	}
}
