package accounting_test

import (
	"appengine/aetest"
	"appengine/datastore"
	"encoding/gob"
	_ "fmt"
	"github.com/mcesarhm/geek-accounting/go-server/accounting"
	"github.com/mcesarhm/geek-accounting/go-server/accounting/reporting"
	"github.com/mcesarhm/geek-accounting/go-server/cache"
	"github.com/mcesarhm/geek-accounting/go-server/context"
	"github.com/mcesarhm/geek-accounting/go-server/core"
	"github.com/mcesarhm/geek-accounting/go-server/db"
	"testing"
	"time"
)

func init() {
	gob.Register(([]*accounting.Account)(nil))
	gob.Register((*accounting.Account)(nil))
	gob.Register(([]*accounting.Transaction)(nil))
	gob.Register(([]interface{})(nil))
}

func newContext(c *context.Context) (aetest.Context, error) {
	ac, err := aetest.NewContext(nil)
	if err != nil {
		return nil, err
	}
	c.Db = db.NewAppengineDb(ac)
	c.Cache = cache.NewAppengineCache(ac)
	return ac, nil
}

func TestSaveChartOfAccounts(t *testing.T) {
	c := context.Context{}
	ac, err := newContext(&c)
	if err != nil {
		t.Fatal(err)
	}
	defer ac.Close()
	var coa *accounting.ChartOfAccounts
	if coa, err = saveChartOfAccounts(c); err != nil {
		t.Fatal(err)
	}
	if coa.Key.IsNil() {
		t.Error("Key must not be null")
	}
	if coa.Name != "coa" {
		t.Errorf("Name (%v) must be 'coa'", coa.Name)
	}
	time.Sleep(100 * time.Millisecond)
	var obj interface{}
	if obj, err = accounting.AllChartsOfAccounts(c, nil, core.NewNilUserKey()); err != nil {
		t.Fatal(err)
	}
	coas := *obj.(*[]accounting.ChartOfAccounts)
	if len(coas) == 0 {
		t.Error("The chart of accounts must be persisted")
	}
}

func TestSaveAccount(t *testing.T) {
	c := context.Context{}
	ac, err := newContext(&c)
	if err != nil {
		t.Fatal(err)
	}
	defer ac.Close()
	var coa *accounting.ChartOfAccounts
	if coa, err = saveChartOfAccounts(c); err != nil {
		t.Fatal(err)
	}
	var a *accounting.Account
	if a, err = saveAccount(c, coa, "1", "a1", []string{"balanceSheet", "debitBalance"}); err != nil {
		t.Fatal(err)
	}
	if a.Key.IsNil() {
		t.Error("Key must not be null")
	}
	if a.Number != "1" {
		t.Errorf("The number (%v) must be 1", a.Number)
	}
	if a.Name != "a1" {
		t.Errorf("The name (%v) must be 'account'", a.Name)
	}
	var obj interface{}
	if obj, err = accounting.AllAccounts(c, map[string]string{"coa": coa.Key.Encode()}, core.NewNilUserKey()); err != nil {
		t.Fatal(err)
	}
	accounts := *obj.(*[]accounting.Account)
	if len(accounts) == 0 {
		t.Error("The account must be persisted")
	}
}

func TestSaveTransaction(t *testing.T) {
	c := context.Context{}
	ac, err := newContext(&c)
	if err != nil {
		t.Fatal(err)
	}
	defer ac.Close()

	var coa *accounting.ChartOfAccounts
	if coa, err = saveChartOfAccounts(c); err != nil {
		t.Fatal(err)
	}
	var a1, a2 *accounting.Account
	if a1, err = saveAccount(c, coa, "1", "Assets", []string{"balanceSheet", "debitBalance"}); err != nil {
		t.Fatal(err)
	}
	if a2, err = saveAccount(c, coa, "2", "Liabilities", []string{"balanceSheet", "creditBalance"}); err != nil {
		t.Fatal(err)
	}

	var tx *accounting.Transaction
	if tx, err = saveTransaction(c, coa, "1", "2", ""); err != nil {
		t.Fatal(err)
	}

	if tx.Key.IsNil() {
		t.Error("Key must not be null")
	}
	if tx.Debits[0].Account == a1.Key {
		t.Error("Account's key must not be null")
	}
	if tx.Credits[0].Account == a2.Key {
		t.Error("Account's key must not be null")
	}

	var obj interface{}
	if obj, err = accounting.AllTransactions(c, map[string]string{"coa": coa.Key.Encode()}, core.NewNilUserKey()); err != nil {
		t.Fatal(err)
	}
	transactions := *obj.(*[]accounting.Transaction)
	if len(transactions) == 0 {
		t.Error("The transaction must be persisted")
	}
}

func TestJournal(t *testing.T) {
	c := context.Context{}
	ac, err := newContext(&c)
	if err != nil {
		t.Fatal(err)
	}
	defer ac.Close()

	var coa *accounting.ChartOfAccounts
	if coa, err = saveChartOfAccounts(c); err != nil {
		t.Fatal(err)
	}
	if _, err = saveAccount(c, coa, "1", "Assets", []string{"balanceSheet", "debitBalance"}); err != nil {
		t.Fatal(err)
	}
	if _, err = saveAccount(c, coa, "2", "Liabilities", []string{"balanceSheet", "creditBalance"}); err != nil {
		t.Fatal(err)
	}
	var tx1, tx2 *accounting.Transaction
	if tx1, err = saveTransaction(c, coa, "1", "2", ""); err != nil {
		t.Fatal(err)
	}

	var obj interface{}
	if obj, err = reporting.Journal(c, map[string]string{"coa": coa.Key.Encode(), "from": "2014-05-01", "to": "2014-05-01"}, core.NewNilUserKey()); err != nil {
		t.Fatal(err)
	}
	journal := obj.([]map[string]interface{})
	if len(journal) != 1 {
		t.Error("Journal must have one entry")
	}
	if journal[0]["_id"].(*datastore.Key).Encode() != tx1.Key.Encode() {
		t.Error("Journal's entry must encode transaction's key")
	}

	if tx2, err = saveTransaction(c, coa, "2", "1", ""); err != nil {
		t.Fatal(err)
	}
	if obj, err = reporting.Journal(c, map[string]string{"coa": coa.Key.Encode(), "from": "2014-05-01", "to": "2014-05-01"}, core.NewNilUserKey()); err != nil {
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
	if obj, err = reporting.Journal(c, map[string]string{"coa": coa.Key.Encode(), "from": "2014-05-02", "to": "2014-05-02"}, core.NewNilUserKey()); err != nil {
		t.Fatal(err)
	}
	journal = obj.([]map[string]interface{})
	if len(journal) != 0 {
		t.Error("Journal must have no entries")
	}
}

func TestLedger(t *testing.T) {
	c := context.Context{}
	ac, err := newContext(&c)
	if err != nil {
		t.Fatal(err)
	}
	defer ac.Close()

	var coa *accounting.ChartOfAccounts
	if coa, err = saveChartOfAccounts(c); err != nil {
		t.Fatal(err)
	}
	if _, err = saveAccount(c, coa, "1", "Assets", []string{"balanceSheet", "debitBalance"}); err != nil {
		t.Fatal(err)
	}
	if _, err = saveAccount(c, coa, "2", "Liabilities", []string{"balanceSheet", "creditBalance"}); err != nil {
		t.Fatal(err)
	}
	if _, err = saveTransaction(c, coa, "1", "2", ""); err != nil {
		t.Fatal(err)
	}

	var obj interface{}
	if obj, err = reporting.Ledger(c, map[string]string{"coa": coa.Key.Encode(), "from": "2014-05-01", "to": "2014-05-01", "account": "1"}, core.NewNilUserKey()); err != nil {
		t.Fatal(err)
	}
	ledger := obj.(map[string]interface{})
	account := ledger["account"].(map[string]interface{})
	if account["name"] != "Assets" {
		t.Error("Account name must be 'Assets'")
	}
	if len(ledger["entries"].([]interface{})) != 1 {
		t.Error("Ledger must have one entry")
	} else {
		entry := ledger["entries"].([]interface{})[0].(map[string]interface{})
		if entry["counterpart"].(map[string]interface{})["number"] != "2" {
			t.Error("Counterpart must be account #2")
		}
		if entry["balance"] != 1.0 {
			t.Error("Entry's balance must be 1")
		}
	}
	if ledger["balance"] != 0.0 {
		t.Error("Ledger's balance must be 0")
	}

	if obj, err = reporting.Ledger(c, map[string]string{"coa": coa.Key.Encode(), "from": "2014-05-01", "to": "2014-05-01", "account": "1"}, core.NewNilUserKey()); err != nil {
		t.Fatal(err)
	}
	ledger = obj.(map[string]interface{})
	if account["name"] != "Assets" {
		t.Error("Account name must be 'Assets'")
	}
	if len(ledger["entries"].([]interface{})) != 1 {
		t.Error("Ledger must have one entry")
	} else {
		entry := ledger["entries"].([]interface{})[0].(map[string]interface{})
		if entry["counterpart"].(map[string]interface{})["number"] != "2" {
			t.Error("Counterpart must be account #2")
		}
		if entry["balance"] != 1.0 {
			t.Error("Entry's balance must be 1")
		}
	}
	if ledger["balance"] != 0.0 {
		t.Error("Ledger's balance must be 0")
	}

	if obj, err = reporting.Ledger(c, map[string]string{"coa": coa.Key.Encode(), "from": "2014-05-02", "to": "2014-05-02", "account": "1"}, core.NewNilUserKey()); err != nil {
		t.Fatal(err)
	}
	ledger = obj.(map[string]interface{})
	if account["name"] != "Assets" {
		t.Error("Account name must be 'Assets'")
	}
	if len(ledger["entries"].([]interface{})) != 0 {
		t.Error("Ledger must have zero entries")
	}
	if ledger["balance"] != 1.0 {
		t.Errorf("Ledger's balance must be 1, but was %v", ledger["balance"])
	}

	for i := 0; i < 4; i++ {
		if _, err := saveTransaction(c, coa, "1", "2", ""); err != nil {
			t.Fatal(err)
		}
	}
	if obj, err = reporting.Ledger(c, map[string]string{"coa": coa.Key.Encode(), "from": "2014-05-01", "to": "2014-05-02", "account": "1"}, core.NewNilUserKey()); err != nil {
		t.Fatal(err)
	}
	ledger = obj.(map[string]interface{})
	if l := len(ledger["entries"].([]interface{})); l != 5 {
		t.Errorf("Ledger must have five entries and have %v", l)
	}

}

func TestBalance(t *testing.T) {
	c := context.Context{}
	ac, err := newContext(&c)
	if err != nil {
		t.Fatal(err)
	}
	defer ac.Close()

	var coa *accounting.ChartOfAccounts
	if coa, err = saveChartOfAccounts(c); err != nil {
		t.Fatal(err)
	}
	var a1, a2 *accounting.Account
	if a1, err = saveAccount(c, coa, "1", "Assets", []string{"balanceSheet", "debitBalance"}); err != nil {
		t.Fatal(err)
	}
	if a2, err = saveAccount(c, coa, "2", "Liabilities", []string{"balanceSheet", "creditBalance"}); err != nil {
		t.Fatal(err)
	}

	if _, err = saveTransaction(c, coa, "1", "2", ""); err != nil {
		t.Fatal(err)
	}
	var obj interface{}
	if obj, err = reporting.Balance(c, map[string]string{"coa": coa.Key.Encode(), "at": "2014-05-01"}, core.NewNilUserKey()); err != nil {
		t.Fatal(err)
	}
	balance := obj.([]db.M)
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

	var tx *accounting.Transaction
	if tx, err = saveTransaction(c, coa, "2", "1", ""); err != nil {
		t.Fatal(err)
	}
	if obj, err = reporting.Balance(c, map[string]string{"coa": coa.Key.Encode(), "at": "2014-05-01"}, core.NewNilUserKey()); err != nil {
		t.Fatal(err)
	}
	balance = obj.([]db.M)
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
	if tx, err = saveTransaction(c, coa, "1", "2", tx.Key.Encode()); err != nil {
		t.Fatal(err)
	}
	if obj, err = reporting.Balance(c, map[string]string{"coa": coa.Key.Encode(), "at": "2014-05-01"}, core.NewNilUserKey()); err != nil {
		t.Fatal(err)
	}
	balance = obj.([]db.M)
	if len(balance) != 2 {
		t.Error("Balance must have two entries")
	}
	if balance[0]["account"].(map[string]interface{})["number"] != a1.Number {
		t.Error("Balance's entry must have account number")
	}
	if balance[1]["account"].(map[string]interface{})["number"] != a2.Number {
		t.Error("Balance's entry must have account number")
	}
	if balance[0]["value"] != 2.0 {
		t.Error("Balance's value must be 2")
	}
	if balance[1]["value"] != 2.0 {
		t.Error("Balance's value must be 2")
	}
	if err = c.Cache.Flush(); err != nil {
		t.Fatal(err)
	}
	if tx, err = saveTransaction(c, coa, "2", "1", tx.Key.Encode()); err != nil {
		t.Fatal(err)
	}
	if obj, err = reporting.Balance(c, map[string]string{"coa": coa.Key.Encode(), "at": "2014-05-01"}, core.NewNilUserKey()); err != nil {
		t.Fatal(err)
	}
	balance = obj.([]db.M)
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
	if _, err = accounting.DeleteTransaction(c, nil, map[string]string{"transaction": tx.Key.Encode()}, core.NewNilUserKey()); err != nil {
		t.Fatal(err)
	}
	if obj, err = reporting.Balance(c, map[string]string{"coa": coa.Key.Encode(), "at": "2014-05-01"}, core.NewNilUserKey()); err != nil {
		t.Fatal(err)
	}
	balance = obj.([]db.M)
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

func saveChartOfAccounts(c context.Context) (*accounting.ChartOfAccounts, error) {
	if obj, err := accounting.SaveChartOfAccounts(c, map[string]interface{}{"name": "coa"}, nil, core.NewNilUserKey()); err != nil {
		return nil, err
	} else {
		return obj.(*accounting.ChartOfAccounts), nil
	}
	return nil, nil
}

func saveAccount(c context.Context, coa *accounting.ChartOfAccounts, number, name string, tags []string) (*accounting.Account, error) {
	m := map[string]interface{}{"number": number, "name": name}
	for _, t := range tags {
		m[t] = true
	}
	if obj, err := accounting.SaveAccount(c, m, map[string]string{"coa": coa.Key.Encode()}, core.NewNilUserKey()); err != nil {
		return nil, err
	} else {
		return obj.(*accounting.Account), nil
	}
	return nil, nil
}

func saveTransaction(c context.Context, coa *accounting.ChartOfAccounts, a1, a2, tx string) (*accounting.Transaction, error) {
	txMap := map[string]interface{}{
		"debits":  []interface{}{map[string]interface{}{"account": a1, "value": 1.0}},
		"credits": []interface{}{map[string]interface{}{"account": a2, "value": 1.0}},
		"memo":    "test", "date": "2014-05-01T00:00:00Z"}
	param := map[string]string{"coa": coa.Key.Encode()}
	if len(tx) > 0 {
		param["transaction"] = tx
	}
	if obj, err := accounting.SaveTransaction(c, txMap, param, core.NewNilUserKey()); err != nil {
		return nil, err
	} else {
		return obj.(*accounting.Transaction), nil
	}
	return nil, nil
}
