package accounting

import (
	_ "fmt"
	"github.com/mcesarhm/geek-accounting/go-server/context"
	"github.com/mcesarhm/geek-accounting/go-server/core"
	"testing"
	"time"
)

func TestSaveChartOfAccounts(t *testing.T) {
	c := context.Context{}
	ac, err := context.NewContext(&c)
	if err != nil {
		t.Fatal(err)
	}
	defer ac.Close()
	var coa *ChartOfAccounts
	if coa, err = SaveChartOfAccountsSample(c); err != nil {
		t.Fatal(err)
	}
	if coa.Key.IsZero() {
		t.Error("Key must not be null")
	}
	if coa.Name != "coa" {
		t.Errorf("Name (%v) must be 'coa'", coa.Name)
	}
	time.Sleep(100 * time.Millisecond)
	var obj interface{}
	if obj, err = AllChartsOfAccounts(c, nil, core.NewUserKey()); err != nil {
		t.Fatal(err)
	}
	coas := *obj.(*[]ChartOfAccounts)
	if len(coas) == 0 {
		t.Error("The chart of accounts must be persisted")
	}
}

func TestSaveAccount(t *testing.T) {
	c := context.Context{}
	ac, err := context.NewContext(&c)
	if err != nil {
		t.Fatal(err)
	}
	defer ac.Close()
	var coa *ChartOfAccounts
	if coa, err = SaveChartOfAccountsSample(c); err != nil {
		t.Fatal(err)
	}
	var a *Account
	if a, err = SaveAccountSample(c, coa, "1", "a1", []string{"balanceSheet", "debitBalance"}); err != nil {
		t.Fatal(err)
	}
	if a.Key.IsZero() {
		t.Error("Key must not be null")
	}
	if a.Number != "1" {
		t.Errorf("The number (%v) must be 1", a.Number)
	}
	if a.Name != "a1" {
		t.Errorf("The name (%v) must be 'account'", a.Name)
	}
	var obj interface{}
	if obj, err = AllAccounts(c, map[string]string{"coa": coa.Key.Encode()}, core.NewUserKey()); err != nil {
		t.Fatal(err)
	}
	accounts := *obj.(*[]Account)
	if len(accounts) == 0 {
		t.Error("The account must be persisted")
	}
}

func TestSaveTransaction(t *testing.T) {
	c := context.Context{}
	ac, err := context.NewContext(&c)
	if err != nil {
		t.Fatal(err)
	}
	defer ac.Close()

	var coa *ChartOfAccounts
	if coa, err = SaveChartOfAccountsSample(c); err != nil {
		t.Fatal(err)
	}
	var a1, a2 *Account
	if a1, err = SaveAccountSample(c, coa, "1", "Assets", []string{"balanceSheet", "debitBalance"}); err != nil {
		t.Fatal(err)
	}
	if a2, err = SaveAccountSample(c, coa, "2", "Liabilities", []string{"balanceSheet", "creditBalance"}); err != nil {
		t.Fatal(err)
	}

	var tx *Transaction
	if tx, err = SaveTransactionSample(c, coa, "1", "2", ""); err != nil {
		t.Fatal(err)
	}

	if tx.Key.IsZero() {
		t.Error("Key must not be null")
	}
	if tx.Debits[0].Account.String() != a1.Key.String() {
		t.Error(tx.Debits[0].Account, "expected but was", a1.Key)
	}
	if tx.Credits[0].Account.String() != a2.Key.String() {
		t.Error(tx.Credits[0].Account, "expected but was", a2.Key)
	}

	var obj interface{}
	if obj, err = AllTransactions(c, map[string]string{"coa": coa.Key.Encode()}, core.NewUserKey()); err != nil {
		t.Fatal(err)
	}
	transactions := *obj.(*[]Transaction)
	if len(transactions) == 0 {
		t.Error("The transaction must be persisted")
	}
}
