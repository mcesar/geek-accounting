package accounting

import (
	"appengine"
	"appengine/aetest"
	"appengine/datastore"
	"encoding/gob"
	_ "fmt"
	"testing"
	"time"
)

func init() {
	gob.Register(([]*Account)(nil))
	gob.Register((*Account)(nil))
}

func TestSaveChartOfAccounts(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	var coa *ChartOfAccounts
	if coa, err = saveChartOfAccounts(c); err != nil {
		t.Fatal(err)
	}
	if coa.Key == nil {
		t.Error("Key must not be null")
	}
	if coa.Name != "coa" {
		t.Errorf("Name (%v) must be 'coa'", coa.Name)
	}
	time.Sleep(100 * time.Millisecond)
	var obj interface{}
	if obj, err = AllChartsOfAccounts(c, nil, nil); err != nil {
		t.Fatal(err)
	}
	coas := *obj.(*[]ChartOfAccounts)
	if len(coas) == 0 {
		t.Error("The chart of accounts must be persisted")
	}
}

func TestSaveAccount(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	var coa *ChartOfAccounts
	if coa, err = saveChartOfAccounts(c); err != nil {
		t.Fatal(err)
	}
	var a *Account
	if a, err = saveAccount(c, coa, "1", "a1", []string{"balanceSheet", "debitBalance"}); err != nil {
		t.Fatal(err)
	}
	if a.Key == nil {
		t.Error("Key must not be null")
	}
	if a.Number != "1" {
		t.Errorf("The number (%v) must be 1", a.Number)
	}
	if a.Name != "a1" {
		t.Errorf("The name (%v) must be 'account'", a.Name)
	}
	var obj interface{}
	if obj, err = AllAccounts(c, map[string]string{"coa": coa.Key.Encode()}, nil); err != nil {
		t.Fatal(err)
	}
	accounts := *obj.(*[]Account)
	if len(accounts) == 0 {
		t.Error("The account must be persisted")
	}
}

func TestSaveTransaction(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	var coa *ChartOfAccounts
	if coa, err = saveChartOfAccounts(c); err != nil {
		t.Fatal(err)
	}
	var a1, a2 *Account
	if a1, err = saveAccount(c, coa, "1", "Assets", []string{"balanceSheet", "debitBalance"}); err != nil {
		t.Fatal(err)
	}
	if a2, err = saveAccount(c, coa, "2", "Liabilities", []string{"balanceSheet", "creditBalance"}); err != nil {
		t.Fatal(err)
	}

	var tx *Transaction
	if tx, err = saveTransaction(c, coa, "1", "2"); err != nil {
		t.Fatal(err)
	}

	if tx.Key == nil {
		t.Error("Key must not be null")
	}
	if tx.Debits[0].Account == a1.Key {
		t.Error("Account's key must not be null")
	}
	if tx.Credits[0].Account == a2.Key {
		t.Error("Account's key must not be null")
	}

	var obj interface{}
	if obj, err = AllTransactions(c, map[string]string{"coa": coa.Key.Encode()}, nil); err != nil {
		t.Fatal(err)
	}
	transactions := *obj.(*[]Transaction)
	if len(transactions) == 0 {
		t.Error("The transaction must be persisted")
	}
}

func TestJournal(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	var coa *ChartOfAccounts
	if coa, err = saveChartOfAccounts(c); err != nil {
		t.Fatal(err)
	}
	if _, err = saveAccount(c, coa, "1", "Assets", []string{"balanceSheet", "debitBalance"}); err != nil {
		t.Fatal(err)
	}
	if _, err = saveAccount(c, coa, "2", "Liabilities", []string{"balanceSheet", "creditBalance"}); err != nil {
		t.Fatal(err)
	}
	var tx1, tx2 *Transaction
	if tx1, err = saveTransaction(c, coa, "1", "2"); err != nil {
		t.Fatal(err)
	}

	var obj interface{}
	if obj, err = Journal(c, map[string]string{"coa": coa.Key.Encode(), "from": "2014-05-01", "to": "2014-05-01"}, nil); err != nil {
		t.Fatal(err)
	}
	journal := obj.([]map[string]interface{})
	if len(journal) != 1 {
		t.Error("Journal must have one entry")
	}
	if journal[0]["_id"].(*datastore.Key).Encode() != tx1.Key.Encode() {
		t.Error("Journal's entry must encode transaction's key")
	}

	if tx2, err = saveTransaction(c, coa, "2", "1"); err != nil {
		t.Fatal(err)
	}
	if obj, err = Journal(c, map[string]string{"coa": coa.Key.Encode(), "from": "2014-05-01", "to": "2014-05-01"}, nil); err != nil {
		t.Fatal(err)
	}
	journal = obj.([]map[string]interface{})
	if len(journal) != 2 {
		t.Error("Journal must have two entries")
	}
	if journal[0]["_id"].(*datastore.Key).Encode() != tx1.Key.Encode() {
		t.Error("Journal's entry must encode transaction's key")
	}
	if journal[1]["_id"].(*datastore.Key).Encode() != tx2.Key.Encode() {
		t.Error("Journal's entry must encode transaction's key")
	}
}

func TestBalance(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	var coa *ChartOfAccounts
	if coa, err = saveChartOfAccounts(c); err != nil {
		t.Fatal(err)
	}
	var a1, a2 *Account
	if a1, err = saveAccount(c, coa, "1", "Assets", []string{"balanceSheet", "debitBalance"}); err != nil {
		t.Fatal(err)
	}
	if a2, err = saveAccount(c, coa, "2", "Liabilities", []string{"balanceSheet", "creditBalance"}); err != nil {
		t.Fatal(err)
	}

	if _, err = saveTransaction(c, coa, "1", "2"); err != nil {
		t.Fatal(err)
	}
	var obj interface{}
	if obj, err = Balance(c, map[string]string{"coa": coa.Key.Encode(), "at": "2014-05-01"}, nil); err != nil {
		t.Fatal(err)
	}
	balance := obj.([]map[string]interface{})
	if len(balance) != 2 {
		t.Error("Balance must have two entries")
	}
	if balance[0]["account"].(map[string]interface{})["number"] != a1.Number {
		t.Error("Balance's entry must have account number")
	}
	if balance[1]["account"].(map[string]interface{})["number"] != a2.Number {
		t.Error("Balance's entry must have account number")
	}
	if balance[0]["value"] != 1.0 {
		t.Error("Balance's value must be 1")
	}
	if balance[1]["value"] != 1.0 {
		t.Error("Balance's value must be 1")
	}

	var tx *Transaction
	if tx, err = saveTransaction(c, coa, "2", "1"); err != nil {
		t.Fatal(err)
	}
	if obj, err = Balance(c, map[string]string{"coa": coa.Key.Encode(), "at": "2014-05-01"}, nil); err != nil {
		t.Fatal(err)
	}
	balance = obj.([]map[string]interface{})
	if len(balance) != 2 {
		t.Error("Balance must have two entries")
	}
	if balance[0]["account"].(map[string]interface{})["number"] != a1.Number {
		t.Error("Balance's entry must have account number")
	}
	if balance[1]["account"].(map[string]interface{})["number"] != a2.Number {
		t.Error("Balance's entry must have account number")
	}
	if balance[0]["value"] != 0.0 {
		t.Error("Balance's value must be 0")
	}
	if balance[1]["value"] != 0.0 {
		t.Error("Balance's value must be 0")
	}
	if _, err = DeleteTransaction(c, nil, map[string]string{"transaction": tx.Key.Encode()}, nil); err != nil {
		t.Fatal(err)
	}
	if obj, err = Balance(c, map[string]string{"coa": coa.Key.Encode(), "at": "2014-05-01"}, nil); err != nil {
		t.Fatal(err)
	}
	balance = obj.([]map[string]interface{})
	if len(balance) != 2 {
		t.Error("Balance must have two entries")
	}
	if balance[0]["account"].(map[string]interface{})["number"] != a1.Number {
		t.Error("Balance's entry must have account number")
	}
	if balance[1]["account"].(map[string]interface{})["number"] != a2.Number {
		t.Error("Balance's entry must have account number")
	}
	if balance[0]["value"] != 1.0 {
		t.Error("Balance's value must be 1")
	}
	if balance[1]["value"] != 1.0 {
		t.Error("Balance's value must be 1")
	}
}

func saveChartOfAccounts(c appengine.Context) (*ChartOfAccounts, error) {
	if obj, err := SaveChartOfAccounts(c, map[string]interface{}{"name": "coa"}, nil, nil); err != nil {
		return nil, err
	} else {
		return obj.(*ChartOfAccounts), nil
	}
	return nil, nil
}

func saveAccount(c appengine.Context, coa *ChartOfAccounts, number, name string, tags []string) (*Account, error) {
	m := map[string]interface{}{"number": number, "name": name}
	for _, t := range tags {
		m[t] = true
	}
	if obj, err := SaveAccount(c, m, map[string]string{"coa": coa.Key.Encode()}, nil); err != nil {
		return nil, err
	} else {
		return obj.(*Account), nil
	}
	return nil, nil
}

func saveTransaction(c appengine.Context, coa *ChartOfAccounts, a1, a2 string) (*Transaction, error) {
	if obj, err := SaveTransaction(c, map[string]interface{}{
		"debits":  []interface{}{map[string]interface{}{"account": a1, "value": 1.0}},
		"credits": []interface{}{map[string]interface{}{"account": a2, "value": 1.0}},
		"memo":    "test", "date": "2014-05-01T00:00:00Z"},
		map[string]string{"coa": coa.Key.Encode()}, nil); err != nil {
		return nil, err
	} else {
		return obj.(*Transaction), nil
	}
	return nil, nil
}
