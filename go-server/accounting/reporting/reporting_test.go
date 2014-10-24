package reporting

import (
	_ "fmt"
	"github.com/mcesarhm/geek-accounting/go-server/accounting"
	"github.com/mcesarhm/geek-accounting/go-server/context"
	"github.com/mcesarhm/geek-accounting/go-server/core"
	"github.com/mcesarhm/geek-accounting/go-server/db"
	"testing"
)

func TestJournal(t *testing.T) {
	c := context.Context{}
	ac, err := context.NewContext(&c)
	if err != nil {
		t.Fatal(err)
	}
	defer ac.Close()

	var coa *accounting.ChartOfAccounts
	if coa, err = accounting.SaveChartOfAccountsSample(c); err != nil {
		t.Fatal(err)
	}
	if _, err = accounting.SaveAccountSample(c, coa, "1", "Assets", []string{"balanceSheet", "debitBalance"}); err != nil {
		t.Fatal(err)
	}
	if _, err = accounting.SaveAccountSample(c, coa, "2", "Liabilities", []string{"balanceSheet", "creditBalance"}); err != nil {
		t.Fatal(err)
	}
	var tx1, tx2 *accounting.Transaction
	if tx1, err = accounting.SaveTransactionSample(c, coa, "1", "2", ""); err != nil {
		t.Fatal(err)
	}

	var obj interface{}
	if obj, err = Journal(c, map[string]string{"coa": coa.Key.Encode(), "from": "2014-05-01", "to": "2014-05-01"}, core.NewUserKey()); err != nil {
		t.Fatal(err)
	}
	journal := obj.([]map[string]interface{})
	if len(journal) != 1 {
		t.Error("Journal must have one entry")
	}
	if journal[0]["_id"].(db.Key).Encode() != tx1.Key.Encode() {
		t.Error("Journal's entry must encode transaction's key")
	}

	if tx2, err = accounting.SaveTransactionSample(c, coa, "2", "1", ""); err != nil {
		t.Fatal(err)
	}
	if obj, err = Journal(c, map[string]string{"coa": coa.Key.Encode(), "from": "2014-05-01", "to": "2014-05-01"}, core.NewUserKey()); err != nil {
		t.Fatal(err)
	}
	journal = obj.([]map[string]interface{})
	if len(journal) != 2 {
		t.Error("Journal must have two entries")
	}
	if journal[0]["_id"].(db.Key).Encode() != tx1.Key.Encode() {
		t.Error("Journal's entry must encode transaction's key")
	}
	if journal[1]["_id"].(db.Key).Encode() != tx2.Key.Encode() {
		t.Error("Journal's entry must encode transaction's key")
	}
	if obj, err = Journal(c, map[string]string{"coa": coa.Key.Encode(), "from": "2014-05-02", "to": "2014-05-02"}, core.NewUserKey()); err != nil {
		t.Fatal(err)
	}
	journal = obj.([]map[string]interface{})
	if len(journal) != 0 {
		t.Error("Journal must have no entries")
	}
}

func TestLedger(t *testing.T) {
	c := context.Context{}
	ac, err := context.NewContext(&c)
	if err != nil {
		t.Fatal(err)
	}
	defer ac.Close()

	var coa *accounting.ChartOfAccounts
	if coa, err = accounting.SaveChartOfAccountsSample(c); err != nil {
		t.Fatal(err)
	}
	if _, err = accounting.SaveAccountSample(c, coa, "1", "Assets", []string{"balanceSheet", "debitBalance"}); err != nil {
		t.Fatal(err)
	}
	if _, err = accounting.SaveAccountSample(c, coa, "2", "Liabilities", []string{"balanceSheet", "creditBalance"}); err != nil {
		t.Fatal(err)
	}
	if _, err = accounting.SaveTransactionSample(c, coa, "1", "2", ""); err != nil {
		t.Fatal(err)
	}

	var obj interface{}
	if obj, err = Ledger(c, map[string]string{"coa": coa.Key.Encode(), "from": "2014-05-01", "to": "2014-05-01", "account": "1"}, core.NewUserKey()); err != nil {
		t.Fatal(err)
	}
	ledger := obj.(map[string]interface{})
	account := ledger["account"].(map[string]interface{})
	if account["name"] != "Assets" {
		t.Error("Account name must be 'Assets'")
	}
	if len(ledger["entries"].([]interface{})) != 1 {
		t.Error("Ledger must have one entry, but was", len(ledger["entries"].([]interface{})))
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

	if obj, err = Ledger(c, map[string]string{"coa": coa.Key.Encode(), "from": "2014-05-01", "to": "2014-05-01", "account": "1"}, core.NewUserKey()); err != nil {
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

	if obj, err = Ledger(c, map[string]string{"coa": coa.Key.Encode(), "from": "2014-05-02", "to": "2014-05-02", "account": "1"}, core.NewUserKey()); err != nil {
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
		if _, err := accounting.SaveTransactionSample(c, coa, "1", "2", ""); err != nil {
			t.Fatal(err)
		}
	}
	if obj, err = Ledger(c, map[string]string{"coa": coa.Key.Encode(), "from": "2014-05-01", "to": "2014-05-02", "account": "1"}, core.NewUserKey()); err != nil {
		t.Fatal(err)
	}
	ledger = obj.(map[string]interface{})
	if l := len(ledger["entries"].([]interface{})); l != 5 {
		t.Errorf("Ledger must have five entries and have %v", l)
	}

}

func TestBalance(t *testing.T) {
	c := context.Context{}
	ac, err := context.NewContext(&c)
	if err != nil {
		t.Fatal(err)
	}
	defer ac.Close()

	var coa *accounting.ChartOfAccounts
	if coa, err = accounting.SaveChartOfAccountsSample(c); err != nil {
		t.Fatal(err)
	}
	var a1, a2 *accounting.Account
	if a1, err = accounting.SaveAccountSample(c, coa, "1", "Assets", []string{"balanceSheet", "debitBalance"}); err != nil {
		t.Fatal(err)
	}
	if a2, err = accounting.SaveAccountSample(c, coa, "2", "Liabilities", []string{"balanceSheet", "creditBalance"}); err != nil {
		t.Fatal(err)
	}

	if _, err = accounting.SaveTransactionSample(c, coa, "1", "2", ""); err != nil {
		t.Fatal(err)
	}
	var obj interface{}
	if obj, err = Balance(c, map[string]string{"coa": coa.Key.Encode(), "at": "2014-05-01"}, core.NewUserKey()); err != nil {
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
	if tx, err = accounting.SaveTransactionSample(c, coa, "2", "1", ""); err != nil {
		t.Fatal(err)
	}
	if obj, err = Balance(c, map[string]string{"coa": coa.Key.Encode(), "at": "2014-05-01"}, core.NewUserKey()); err != nil {
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
	if tx, err = accounting.SaveTransactionSample(c, coa, "1", "2", tx.Key.Encode()); err != nil {
		t.Fatal(err)
	}
	if obj, err = Balance(c, map[string]string{"coa": coa.Key.Encode(), "at": "2014-05-01"}, core.NewUserKey()); err != nil {
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
		t.Error("Balance's value must be 2, but was", balance[0]["value"])
	}
	if balance[1]["value"] != 2.0 {
		t.Error("Balance's value must be 2, but was", balance[1]["value"])
	}
	if err = c.Cache.Flush(); err != nil {
		t.Fatal(err)
	}
	if tx, err = accounting.SaveTransactionSample(c, coa, "2", "1", tx.Key.Encode()); err != nil {
		t.Fatal(err)
	}
	if obj, err = Balance(c, map[string]string{"coa": coa.Key.Encode(), "at": "2014-05-01"}, core.NewUserKey()); err != nil {
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
	if _, err = accounting.DeleteTransaction(c, nil, map[string]string{"transaction": tx.Key.Encode()}, core.NewUserKey()); err != nil {
		t.Fatal(err)
	}
	if obj, err = Balance(c, map[string]string{"coa": coa.Key.Encode(), "at": "2014-05-01"}, core.NewUserKey()); err != nil {
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
