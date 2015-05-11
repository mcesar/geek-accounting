package deb

import (
	"testing"
)

func TestNewSmallSpace(t *testing.T) {
	s := NewSmallSpace(Array{{{1}},{{-1}}})
	for tx := range s.Transactions() {
		if tx.Moment != Moment(1) {
			t.Errorf("Moment 1 != %v", tx.Moment)
		}
		if tx.Date != Date(1) {
			t.Errorf("Date 1 != %v", tx.Date)
		}
		if len(tx.Entries) != 2 {
			t.Errorf("#Entries 2 != %v", len(tx.Entries))
		}
		if tx.Entries[Account(1)] != 1 {
			t.Errorf("Entry[1] != %v", tx.Entries[Account(1)])
		}
		if tx.Entries[Account(2)] != -1 {
			t.Errorf("Entry[2] != %v", tx.Entries[Account(2)])
		}
	}
}