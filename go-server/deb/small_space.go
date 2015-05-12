package deb

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

func (ss *smallSpace) Append(s Space) {
	if other_ss, ok := s.(*smallSpace); ok {
		ss.arr.Append(other_ss.arr, other_ss.dateOffset, other_ss.momentOffset)
	}
}

func (ss *smallSpace) Slice(a []Account, d []DateRange, m []MomentRange) Space {
	return nil
}

func (ss *smallSpace) Projection(a []Account, d []DateRange, m []MomentRange) Space {
	return nil
}

func (ss *smallSpace) Transactions() chan *Transaction {
	out := make(chan *Transaction)
	go func() {
		if !ss.arr.Empty() {
			x, y, z := ss.arr.Dimensions()
			for k := 0; k < z; k++ {
				for j := 0; j < y; j++ {
					t := Transaction{Moment(k + 1 + ss.momentOffset), Date(j + 1 + ss.dateOffset), make(Entries)}
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
		}
		close(out)
	}()
	return out
}
