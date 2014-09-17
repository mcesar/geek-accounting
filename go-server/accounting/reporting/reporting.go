package reporting

import (
	"appengine"
	"appengine/datastore"
	"fmt"
	"github.com/mcesarhm/geek-accounting/go-server/accounting"
	"github.com/mcesarhm/geek-accounting/go-server/util"
	"math"
	"strings"
	"time"
)

func Balance(c appengine.Context, m map[string]string, _ *datastore.Key) (result interface{}, err error) {
	from := time.Date(1000, 1, 1, 0, 0, 0, 0, time.UTC)
	to, err := time.Parse(time.RFC3339, m["at"]+"T00:00:00Z")
	if err != nil {
		return
	}
	b, err := accounting.Balances(c, m["coa"], from, to, map[string]interface{}{"Tags =": "balanceSheet"})
	if err != nil {
		return
	}
	for _, e := range b {
		e["account"] = accountToMap(e["account"].(*accounting.Account))
	}
	result = b
	return
}

func Journal(c appengine.Context, m map[string]string, _ *datastore.Key) (result interface{}, err error) {

	from, err := time.Parse(time.RFC3339, m["from"]+"T00:00:00Z")
	if err != nil {
		return
	}
	to, err := time.Parse(time.RFC3339, m["to"]+"T00:00:00Z")
	if err != nil {
		return
	}

	accountKeys, accounts, err := accounting.Accounts(c, m["coa"], nil)
	if err != nil {
		return
	}

	transactionKeys, transactions, err := accounting.Transactions(c, m["coa"], map[string]interface{}{"Date >=": from, "Date <=": to})
	if err != nil {
		return
	}

	accountsMap := map[string]*accounting.Account{}
	for i, a := range accounts {
		accountsMap[accountKeys[i].String()] = a
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

func Ledger(c appengine.Context, m map[string]string, _ *datastore.Key) (result interface{}, err error) {

	from, err := time.Parse(time.RFC3339, m["from"]+"T00:00:00Z")
	if err != nil {
		return
	}
	to, err := time.Parse(time.RFC3339, m["to"]+"T00:00:00Z")
	if err != nil {
		return
	}

	accountKeys, accounts, err := accounting.Accounts(c, m["coa"], nil)
	if err != nil {
		return
	}

	var account *accounting.Account
	accountsMap := map[string]*accounting.Account{}
	for i, a := range accounts {
		if a.Number == m["account"] || accountKeys[i].Encode() == m["account"] {
			account = a
			account.Key = accountKeys[i]
		}
		accountsMap[accountKeys[i].String()] = a
	}
	if account == nil {
		err = fmt.Errorf("Account not found")
		return
	}

	transactions, balance, err := accounting.TransactionsWithValue(c, m["coa"], account, from, to)
	if err != nil {
		return
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
		"account": accountToMap(account),
		"entries": resultEntries,
		"balance": balance,
	}

	return
}

func IncomeStatement(c appengine.Context, m map[string]string, _ *datastore.Key) (result interface{}, err error) {
	from, err := time.Parse(time.RFC3339, m["from"]+"T00:00:00Z")
	if err != nil {
		return
	}
	to, err := time.Parse(time.RFC3339, m["to"]+"T00:00:00Z")
	if err != nil {
		return
	}

	balances, err := accounting.Balances(c, m["coa"], from, to, map[string]interface{}{"Tags =": "incomeStatement"})
	if err != nil {
		return
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
		if util.Contains(balance["account"].(*accounting.Account).Tags, "analytic") && balance["value"].(float64) > 0 {
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

	collectRoots := func(parent *datastore.Key) {
		for _, m := range balances {
			account := m["account"].(*accounting.Account)
			if (account.Parent == nil && parent == nil) || account.Parent.String() == parent.String() {
				if util.Contains(account.Tags, "creditBalance") {
					revenueRoots = append(revenueRoots, account)
				} else {
					expenseRoots = append(expenseRoots, account)
				}
			}
		}
	}

	collectRoots(nil)

	if (len(revenueRoots) + len(expenseRoots)) == 1 {
		parentKey := append(revenueRoots, expenseRoots...)[0].Key
		revenueRoots = revenueRoots[0:0]
		expenseRoots = expenseRoots[0:0]
		collectRoots(parentKey)
	}

	for _, m := range balances {
		account := m["account"].(*accounting.Account)
		if util.Contains(account.Tags, "operating") && isDescendent(account, revenueRoots) {
			resultTyped.GrossRevenue = addBalance(resultTyped.GrossRevenue, m)
		} else if util.Contains(account.Tags, "deduction") {
			resultTyped.Deduction = addBalance(resultTyped.Deduction, m)
		} else if util.Contains(account.Tags, "salesTax") {
			resultTyped.SalesTax = addBalance(resultTyped.SalesTax, m)
		} else if util.Contains(account.Tags, "cost") {
			resultTyped.Cost = addBalance(resultTyped.Cost, m)
		} else if util.Contains(account.Tags, "operating") && isDescendent(account, expenseRoots) {
			resultTyped.OperatingExpense = addBalance(resultTyped.OperatingExpense, m)
		} else if util.Contains(account.Tags, "nonOperatingTax") {
			resultTyped.NonOperatingTax = addBalance(resultTyped.NonOperatingTax, m)
		} else if util.Contains(account.Tags, "incomeTax") {
			resultTyped.IncomeTax = addBalance(resultTyped.IncomeTax, m)
		} else if util.Contains(account.Tags, "dividends") {
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

	if resultTyped.NetRevenue.Balance == 0 || (z(resultTyped.Deduction).Balance == 0 && z(resultTyped.SalesTax).Balance == 0) {
		resultTyped.NetRevenue = nil
	}
	if resultTyped.GrossProfit.Balance == 0 || z(resultTyped.Cost).Balance == 0 {
		resultTyped.GrossProfit = nil
	}
	if resultTyped.NetOperatingIncome.Balance == 0 {
		resultTyped.NetOperatingIncome = nil
	}
	if resultTyped.IncomeBeforeIncomeTax.Balance == 0 || z(resultTyped.NonOperatingTax).Balance == 0 {
		resultTyped.IncomeBeforeIncomeTax = nil
	}

	result = resultTyped

	return
}

func accountToMap(account *accounting.Account) map[string]interface{} {
	return map[string]interface{}{
		"_id":           account.Key,
		"number":        account.Number,
		"name":          account.Name,
		"debitBalance":  util.Contains(account.Tags, "debitBalance"),
		"creditBalance": util.Contains(account.Tags, "creditBalance"),
	}
}
