package deb

import (
	"math"

	xmath "github.com/mcesarhm/geek-accounting/go-server/extensions/math"
)

type smallSpace struct {
	arr                      Array
	dateOffset, momentOffset int
}

func NewSmallSpace(arr Array) Space {
	return NewSmallSpaceWithOffset(arr, 0, 0)
}

func NewSmallSpaceWithOffset(arr Array, dateOffset, momentOffset int) Space {
	return &smallSpace{arr.Copy(), dateOffset, momentOffset}
}

func (ss *smallSpace) Append(s Space) error {
	if other_ss, ok := s.(*smallSpace); ok {
		ss.arr.Append(&other_ss.arr, other_ss.dateOffset, other_ss.momentOffset)
		return nil
	} else {
		// Computes the array size
		var minDate, minMoment, maxAccount, maxDate, maxMoment uint64
		minDate, minMoment = math.MaxUint64, math.MaxUint64
		c, errc := s.Transactions()
		for t := range c {
			if t.Date < Date(minDate) {
				minDate = uint64(t.Date)
			}
			if t.Moment < Moment(minMoment) {
				minMoment = uint64(t.Moment)
			}
			for a, _ := range t.Entries {
				if a > Account(maxAccount) {
					maxAccount = uint64(a)
				}
			}
			if t.Date > Date(maxDate) {
				maxDate = uint64(t.Date)
			}
			if t.Moment > Moment(maxMoment) {
				maxMoment = uint64(t.Moment)
			}
		}
		if err := <-errc; err != nil {
			return err
		}
		if maxAccount == 0 || maxDate == 0 || maxMoment == 0 {
			return nil
		}
		// Builds the array
		other_arr := NewArray(int(maxAccount), int(maxDate-minDate), int(maxMoment-minMoment))
		c, errc = s.Transactions()
		for t := range c {
			for a, v := range t.Entries {
				other_arr[a-1][t.Date-1][t.Moment-1] = v
			}
		}
		ss.arr.Append(&other_arr, int(minDate), int(minMoment))
		return <-errc
	}
}

func (ss *smallSpace) Slice(a []Account, d []DateRange, m []MomentRange) (Space, error) {
	result := smallSpace{ss.arr.Copy(), ss.dateOffset, ss.momentOffset}
	x, y, z := ss.arr.Dimensions()
	for i := 0; i < x; i++ {
		for j := 0; j < y; j++ {
			for k := 0; k < z; k++ {
				if !containsDate(d, Date(j+1+ss.dateOffset)) ||
					!containsMoment(m, Moment(k+1+ss.momentOffset)) {
					result.arr[i][j][k] = 0
				}
			}
		}
	}
	for j := 0; j < y; j++ {
		for k := 0; k < z; k++ {
			found := false
			for i := 0; i < x; i++ {
				if containsAccount(a, Account(i+1)) && result.arr[i][j][k] != 0 {
					found = true
					break
				}
			}
			if !found {
				for i := 0; i < x; i++ {
					result.arr[i][j][k] = 0
				}
			}
		}
	}
	return &result, nil
}

func (ss *smallSpace) Projection(a []Account, d []DateRange, m []MomentRange) (Space, error) {
	result := smallSpace{ss.arr.Copy(), ss.dateOffset, ss.momentOffset}
	x, y, z := ss.arr.Dimensions()
	for i := 0; i < x; i++ {
		for _, each_dr := range d {
			for _, each_mr := range m {
				sum := int64(0)
				j_start := xmath.Min(int(each_dr.Start-1-Date(ss.dateOffset)), y)
				j_end := xmath.Min(int(each_dr.End-Date(ss.dateOffset)), y)
				k_start := xmath.Min(int(each_mr.Start-1-Moment(ss.momentOffset)), z)
				k_end := xmath.Min(int(each_mr.End-Moment(ss.momentOffset)), z)
				for j := j_start; j < j_end; j++ {
					for k := k_start; k < k_end; k++ {
						sum += result.arr[i][j][k]
					}
				}
				result.arr[i][j_start][k_start] = sum
			}
		}
	}
	for j := 0; j < y; j++ {
		for k := 0; k < z; k++ {
			found := false
			if containsDateStart(d, Date(j+1+ss.dateOffset)) &&
				containsMomentStart(m, Moment(k+1+ss.momentOffset)) {
				for i := 0; i < x; i++ {
					if containsAccount(a, Account(i+1)) && result.arr[i][j][k] != 0 {
						found = true
						break
					}
				}
			}
			if !found {
				for i := 0; i < x; i++ {
					result.arr[i][j][k] = 0
				}
			}
		}
	}
	return &result, nil
}

func (ss *smallSpace) Transactions() (chan *Transaction, chan error) {
	out := make(chan *Transaction)
	errc := make(chan error, 1)
	go func() {
		defer close(out)
		if ss.arr.Empty() {
			return
		}
		x, y, z := ss.arr.Dimensions()
		for k := 0; k < z; k++ {
			for j := 0; j < y; j++ {
				t := Transaction{Moment(k + 1 + ss.momentOffset),
					Date(j + 1 + ss.dateOffset), make(Entries)}
				for i := 0; i < x; i++ {
					if ss.arr[i][j][k] != 0 {
						t.Entries[Account(i+1)] = ss.arr[i][j][k]
					}
				}
				if len(t.Entries) > 0 {
					out <- &t
				}
			}
		}
		errc <- nil
	}()
	return out, errc
}

func containsAccount(a []Account, elem Account) bool {
	for _, each := range a {
		if each == elem {
			return true
		}
	}
	return false
}

func containsDate(d []DateRange, elem Date) bool {
	for _, each := range d {
		if each.Start <= elem && each.End >= elem {
			return true
		}
	}
	return false
}

func containsMoment(m []MomentRange, elem Moment) bool {
	for _, each := range m {
		if each.Start <= elem && each.End >= elem {
			return true
		}
	}
	return false
}

func containsDateStart(d []DateRange, elem Date) bool {
	for _, each := range d {
		if each.Start == elem {
			return true
		}
	}
	return false
}

func containsMomentStart(m []MomentRange, elem Moment) bool {
	for _, each := range m {
		if each.Start == elem {
			return true
		}
	}
	return false
}
