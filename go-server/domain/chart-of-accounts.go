package domain

import (
	//"log"
	"strings"
	"appengine"
	"appengine/datastore"
)

type ChartOfAccounts struct {
	Key string `json:"_id"`
	Name string `json:"name"`
}

func (coa *ChartOfAccounts) ValidationMessage() string {
	if len(strings.TrimSpace(coa.Name)) == 0 {
		return "The name must be informed"
	}
	return ""
}

type ValidationMessager interface {
	ValidationMessage() string
}

type ValidationError struct {
	ValidationMessage string
}

func (e *ValidationError) Error() string {
	return e.ValidationMessage
}

func AllChartsOfAccounts(c appengine.Context) (interface{}, error) {
	q := datastore.NewQuery("ChartOfAccounts")
	var coas []ChartOfAccounts
	keys, error := q.GetAll(c, &coas)
	if coas == nil {
		coas = make([]ChartOfAccounts, 0)
	}
	for i := 0; i < len(coas); i++ {
		coas[i].Key = keys[i].Encode()
	}
	return coas, error
}

func SaveChartOfAccounts(c appengine.Context, coa *ChartOfAccounts) error {
	vm := coa.ValidationMessage()
	if len(vm) > 0 {
		return &ValidationError{ValidationMessage: vm}
	}
	key, err := datastore.Put(c, datastore.NewIncompleteKey(c, "ChartOfAccounts", nil), coa)
	if err != nil {
		return err
	}

	if key != nil {
		coa.Key = key.Encode()
	}
	return nil
}