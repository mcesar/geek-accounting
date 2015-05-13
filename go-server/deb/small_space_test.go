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
	assertTransactions(t, spaces, cases)
}

func TestAppend(t *testing.T) {
	spaces := []Space{
		NewSmallSpace(Array{{{1, -1}}}.Transposed()),
	}
	arguments := []Space{
		NewSmallSpaceWithOffset(Array{{{2, -2}}}.Transposed(), 0, 1),
	}
	cases := []Space{
		NewSmallSpace(Array{{{1, -1}}, {{2, -2}}}.Transposed()),
	}
	spaces[0].Append(arguments[0])
	assertSpaces(t, spaces, cases)
}

func TestSlice(t *testing.T) {
	spaces := []Space{
		NewSmallSpace(Array{{{1, -1}}, {{2, -2}}}.Transposed()),
	}
	arguments := []struct{a []Account; d []DateRange; m []MomentRange}{
		{[]Account{Account(1),Account(2)},[]DateRange{DateRange{1,1}},[]MomentRange{MomentRange{2,2}}},
	}
	cases := []Space{
		NewSmallSpace(Array{{{0, 0}}, {{2, -2}}}.Transposed()),
	}
	for i := range spaces {
		spaces[i] = spaces[i].Slice(arguments[i].a, arguments[i].d, arguments[i].m)
	}
	assertSpaces(t, spaces, cases)
}

func TestProjection(t *testing.T) {
	spaces := []Space{
		NewSmallSpace(Array{{{1, -1}}, {{2, -2}}}.Transposed()),
	}
	arguments := []struct{a []Account; d []DateRange; m []MomentRange}{
		{[]Account{Account(1),Account(2)},[]DateRange{DateRange{1,1}},[]MomentRange{MomentRange{1,2}}},
	}
	cases := []Space{
		NewSmallSpace(Array{{{3, -3}}, {{0, 0}}}.Transposed()),
	}
	for i := range spaces {
		spaces[i] = spaces[i].Projection(arguments[i].a, arguments[i].d, arguments[i].m)
	}
	assertSpaces(t, spaces, cases)
}

func assertTransactions(t *testing.T, spaces []Space, cases []Transaction) {
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

func assertSpaces(t *testing.T, spaces []Space, cases []Space) {
	for i, s := range spaces {
		if !reflect.DeepEqual(s, cases[i]) {
			t.Errorf("%v != %v", s, cases[i])
		}
	}
}