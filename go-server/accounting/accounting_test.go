package accounting

import (
	"appengine/aetest"
	_ "fmt"
	"testing"
	"time"
)

func TestSaveChartOfAccounts(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	var obj interface{}
	if obj, err = SaveChartOfAccounts(c, map[string]interface{}{"name": "coa"}, nil, nil); err != nil {
		t.Fatal(err)
	}
	coa := obj.(*ChartOfAccounts)
	if coa.Key == nil {
		t.Error("A chave deve ser diferente de nulo")
	}
	if coa.Name != "coa" {
		t.Errorf("A nome (%v) deve ser coa", coa.Name)
	}
	time.Sleep(100 * time.Millisecond)
	if obj, err = AllChartsOfAccounts(c, nil, nil); err != nil {
		t.Fatal(err)
	}
	coas := *obj.(*[]ChartOfAccounts)
	if len(coas) == 0 {
		t.Error("O plano de contas deve ser persistido")
	}
}
