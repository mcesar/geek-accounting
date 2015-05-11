package deb

type smallSpace Array

func NewSmallSpace(arr Array) Space {
	ss := make(smallSpace, len(arr))
	for i, plane := range arr {
		ss[i] = make([][]int64, len(plane))
		for j, col := range plane {
			ss[i][j] = make([]int64, len(col))
			copy(ss[i][j], col)
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
		arr := Array(*ss)
		if len(arr) > 0 && len(arr[0]) > 0 && len(arr[0][0]) > 0 {
			for k := 0; k < len(arr[0][0]); k++ {
				for j := 0; j < len(arr[0]); j++ {
					t := Transaction{Moment(k+1), Date(j+1), make(Entries)}
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