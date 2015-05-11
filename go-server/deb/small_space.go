package deb

type smallSpace struct {
	arr                      Array
	dateOffset, momentOffset int
}

func NewSmallSpace(arr Array) Space {
	return NewSmallSpaceWithOffset(arr, 0, 0)
}

func NewSmallSpaceWithOffset(arr Array, dateOffset, momentOffset int) Space {
	ss := smallSpace{make(Array, len(arr)), dateOffset, momentOffset}
	for i, plane := range arr {
		ss.arr[i] = make([][]int64, len(plane))
		for j, col := range plane {
			ss.arr[i][j] = make([]int64, len(col))
			copy(ss.arr[i][j], col)
		}
	}
	return &ss
}

func (ss *smallSpace) Append(s Space) {
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
		arr := ss.arr
		if len(arr) > 0 && len(arr[0]) > 0 && len(arr[0][0]) > 0 {
			for k := 0; k < len(arr[0][0]); k++ {
				for j := 0; j < len(arr[0]); j++ {
					t := Transaction{Moment(k + 1 + ss.momentOffset), Date(j + 1 + ss.dateOffset), make(Entries)}
					for i := 0; i < len(arr); i++ {
						t.Entries[Account(i+1)] = arr[i][j][k]
					}
					out <- &t
				}
			}
		}
		close(out)
	}()
	return out
}
