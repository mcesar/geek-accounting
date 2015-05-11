package deb

import (
	"fmt"
	"testing"
)

func TestNewSmallSpace(t *testing.T) {
	s := NewSmallSpace(Array{{{1}}, {{-1}}})
	cases := []Transaction{
		Transaction{Moment(1), Date(1), Entries{Account(1): 1, Account(2): -1}},
		Transaction{Moment(3), Date(2), Entries{Account(1): 1, Account(2): -1}},
	}
	for tx := range s.Transactions() {
		if fmt.Sprint(*tx) != fmt.Sprint(cases[0]) {
			t.Errorf("%v != %v", tx, cases[0])
		}
	}
	s = NewSmallSpaceWithOffset(Array{{{1}}, {{-1}}}, 1, 2)
	for tx := range s.Transactions() {
		if fmt.Sprint(*tx) != fmt.Sprint(cases[1]) {
			t.Errorf("%v != %v", tx, cases[1])
		}
	}
}
