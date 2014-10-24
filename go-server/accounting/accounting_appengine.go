// +build appengine

package accounting

import (
	"appengine"
	"appengine/datastore"
)

func UpdateSchema(c appengine.Context) (err error) {
	q := datastore.NewQuery("Transaction")
	transactions := []*Transaction{}
	keys, err := q.GetAll(c, &transactions)
	if err != nil {
		return err
	}
	for i, tx := range transactions {
		if _, err := datastore.Put(c, keys[i], tx); err != nil {
			return err
		}
	}
	return nil
}

func (coa *ChartOfAccounts) Load(c <-chan datastore.Property) (err error) {
	f := make(chan datastore.Property, 32)
	go func() {
		for p := range c {
			if p.Name == "User" || p.Name == "RetainedEarningsAccount" {
				p.Name += ".DsKey"
			}
			f <- p
		}
		close(f)
	}()
	return datastore.LoadStruct(coa, f)
}

func (coa *ChartOfAccounts) Save(c chan<- datastore.Property) error {
	return datastore.SaveStruct(coa, c)
}

func (account *Account) Load(c <-chan datastore.Property) (err error) {
	f := make(chan datastore.Property, 32)
	go func() {
		for p := range c {
			if p.Name == "User" || p.Name == "Parent" {
				p.Name += ".DsKey"
			}
			f <- p
		}
		close(f)
	}()
	return datastore.LoadStruct(account, f)
}

func (account *Account) Save(c chan<- datastore.Property) error {
	return datastore.SaveStruct(account, c)
}

func (transaction *Transaction) Load(c <-chan datastore.Property) (err error) {
	f := make(chan datastore.Property, 32)
	go func() {
		for p := range c {
			if p.Name == "User" || p.Name == "Debits.Account" || p.Name == "Credits.Account" {
				p.Name += ".DsKey"
			}
			f <- p
		}
		close(f)
	}()
	return datastore.LoadStruct(transaction, f)
}

func (transaction *Transaction) Save(c chan<- datastore.Property) error {
	return datastore.SaveStruct(transaction, c)
}
