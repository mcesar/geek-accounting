// +build test

package accounting

import (
	"github.com/mcesarhm/geek-accounting/go-server/context"
	"github.com/mcesarhm/geek-accounting/go-server/core"
)

func SaveChartOfAccountsSample(c context.Context) (*ChartOfAccounts, error) {
	if obj, err := SaveChartOfAccounts(c, map[string]interface{}{"name": "coa"}, nil, core.NewUserKey()); err != nil {
		return nil, err
	} else {
		return obj.(*ChartOfAccounts), nil
	}
	return nil, nil
}

func SaveAccountSample(c context.Context, coa *ChartOfAccounts, number, name string, tags []string) (*Account, error) {
	m := map[string]interface{}{"number": number, "name": name}
	for _, t := range tags {
		m[t] = true
	}
	if obj, err := SaveAccount(c, m, map[string]string{"coa": coa.Key.Encode()}, core.NewUserKey()); err != nil {
		return nil, err
	} else {
		return obj.(*Account), nil
	}
	return nil, nil
}

func SaveTransactionSample(c context.Context, coa *ChartOfAccounts, a1, a2, tx string) (*Transaction, error) {
	txMap := map[string]interface{}{
		"debits":  []interface{}{map[string]interface{}{"account": a1, "value": 1.0}},
		"credits": []interface{}{map[string]interface{}{"account": a2, "value": 1.0}},
		"memo":    "test", "date": "2014-05-01T00:00:00Z"}
	param := map[string]string{"coa": coa.Key.Encode()}
	if len(tx) > 0 {
		param["transaction"] = tx
	}
	if obj, err := SaveTransaction(c, txMap, param, core.NewUserKey()); err != nil {
		return nil, err
	} else {
		return obj.(*Transaction), nil
	}
	return nil, nil
}
