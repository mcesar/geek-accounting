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
	assert(t, spaces, cases)
}

func TestAppend(t *testing.T) {
	spaces := []Space{
		NewSmallSpace(Array{{{1, -1}}}.Transposed()),
		NewSmallSpaceWithOffset(Array{{{2, -2}}}.Transposed(), 0, 1),
		NewSmallSpaceWithOffset(Array{{{1, -1}}, {{2, -2}}}.Transposed(), 0, 0),
	}
	spaces[0].Append(spaces[1])
	if !reflect.DeepEqual(spaces[0], spaces[2]) {
		t.Errorf("%v != %v", spaces[0], spaces[2])
	}
}

func assert(t *testing.T, spaces []Space, cases []Transaction) {
	i := 0
	for _, s := range spaces {
		for tx := range s.Transactions() {
			if !reflect.DeepEqual(*tx, cases[i]) {
				t.Errorf("%v != %v", *tx, cases[i])
			}
			i++
		}
	}
	if i < len(cases) {
		t.Errorf("len(transactions):%v != len(cases):%v", i, len(cases))
	}
}
