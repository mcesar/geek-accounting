package deb

import (
	"reflect"
	"testing"
)

func TestNewSmallSpace(t *testing.T) {
	spaces := []Space{
		NewSmallSpace(Array{{{1, -1}}}.Transposed()),
		NewSmallSpaceWithOffset(Array{{{1, -1}}}.Transposed(), 1, 2),
	}
	cases := []Transaction{
		Transaction{Moment(1), Date(1), Entries{Account(1): 1, Account(2): -1}},
		Transaction{Moment(3), Date(2), Entries{Account(1): 1, Account(2): -1}},
	}
	i := 0
	for _, s := range spaces {
		for tx := range s.Transactions() {
			if !reflect.DeepEqual(*tx, cases[i]) {
				t.Errorf("%v != %v", *tx, cases[i])
			}
			i++
		}
	}
}
