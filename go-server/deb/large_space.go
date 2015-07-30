package deb

import "fmt"

type largeSpace struct {
	blockSize uint
	in        func() chan *dataBlock
	out       chan *dataBlock
	errc      chan error
}

type dataBlock struct {
	key interface{}
	M   []int64
	D   []int32
	A   []int32
	V   []int64
	B   []int16
}

func newLargeSpace(blockSize uint, in func() chan *dataBlock, out chan *dataBlock,
	errc chan error) *largeSpace {
	return &largeSpace{blockSize: blockSize, in: in, out: out, errc: errc}
}

func (ls *largeSpace) Append(s Space) error {
	c, errc := s.Transactions()
	for t := range c {
		if block, err := ls.freeBlock(t); err != nil {
			return err
		} else {
			if block == nil {
				block = ls.newDataBlock()
			}
			block.append(t)
			ls.out <- block
			if err := <-ls.errc; err != nil {
				return err
			}
		}
	}
	if err := <-errc; err != nil {
		return err
	}
	return nil
}

func (ls *largeSpace) Slice(a []Account, d []DateRange, m []MomentRange) (Space, error) {
	out := make(chan *Transaction)
	var err error
	go func() {
		defer close(out)
		err = ls.iterateWithFilter(a, d, m, func(block *dataBlock, i int) {
			out <- block.newTransaction(i)
		})
	}()
	return channelSpace(out), err
}

func (ls *largeSpace) Projection(a []Account, d []DateRange, m []MomentRange) (Space, error) {
	type key struct {
		moment Moment
		date   Date
	}
	transactions := map[key]*Transaction{}
	err := ls.iterateWithFilter(a, d, m, func(block *dataBlock, i int) {
		k := key{startMoment(m, Moment(block.M[i])), startDate(d, Date(block.D[i]))}
		nt := block.newTransaction(i)
		if t, ok := transactions[k]; !ok {
			transactions[k] = nt
		} else {
			for ek, ev := range nt.Entries {
				if ov, ok := t.Entries[ek]; ok {
					t.Entries[ek] = ov + ev
				} else {
					t.Entries[ek] = ev
				}
			}
		}
	})
	if err != nil {
		return nil, err
	}
	out := make(chan *Transaction)
	go func() {
		defer close(out)
		for _, t := range transactions {
			out <- t
		}
	}()
	return channelSpace(out), nil
}

func (ls *largeSpace) Transactions() (chan *Transaction, chan error) {
	out := make(chan *Transaction)
	go func() {
		defer close(out)
		for block := range ls.in() {
			for i := 0; i < len(block.M); i++ {
				out <- block.newTransaction(i)
			}
		}
	}()
	return out, ls.errc
}

func (ls *largeSpace) String() string {
	blocksAsString := []string{}
	count := 0
	for block := range ls.in() {
		blocksAsString = append(blocksAsString, fmt.Sprintf("%v", *block))
		count += 1
	}
	err := <-ls.errc
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("{%v %v %v %v}", ls.blockSize, count, ls.capacity(), blocksAsString)
}

func (block *dataBlock) newTransaction(i int) *Transaction {
	t := Transaction{Moment(block.M[i]), Date(block.D[i]), make(Entries), nil}
	for j := block.B[i*2]; j < block.B[i*2+1]; j++ {
		t.Entries[Account(block.A[j])] = block.V[j]
	}
	return &t
}

func (ls *largeSpace) capacity() uint {
	return ls.blockSize / (64 + 32 + 32*2 + 64*2 + 16*2)
}

func (ls *largeSpace) freeBlock(t *Transaction) (*dataBlock, error) {
	var result *dataBlock
	for block := range ls.in() {
		if uint(len(block.A)+len(t.Entries)) <= ls.capacity()*2 {
			result = block
		}
	}
	return result, <-ls.errc
}

func (ls *largeSpace) newDataBlock() *dataBlock {
	block := new(dataBlock)
	block.M = make([]int64, 0, ls.capacity())
	block.D = make([]int32, 0, ls.capacity())
	block.A = make([]int32, 0, ls.capacity()*2)
	block.V = make([]int64, 0, ls.capacity()*2)
	block.B = make([]int16, 0, ls.capacity()*2)
	return block
}

func (ls *largeSpace) iterateWithFilter(a []Account, d []DateRange, m []MomentRange,
	f func(*dataBlock, int)) error {
	for block := range ls.in() {
		for i := 0; i < len(block.M); i++ {
			if containsMoment(m, Moment(block.M[i])) && containsDate(d, Date(block.D[i])) {
				for j := block.B[i*2]; j < block.B[i*2+1]; j++ {
					if containsAccount(a, Account(block.A[j])) {
						f(block, i)
						break
					}
				}
			}
		}
	}
	return <-ls.errc
}

func (block *dataBlock) append(t *Transaction) {
	mLen := len(block.M)
	aLen := len(block.A)
	block.M = block.M[0 : mLen+1]
	block.D = block.D[0 : mLen+1]
	block.A = block.A[0 : aLen+len(t.Entries)]
	block.V = block.V[0 : aLen+len(t.Entries)]
	block.B = block.B[0 : mLen*2+2]
	block.M[mLen] = int64(t.Moment)
	block.D[mLen] = int32(t.Date)
	i := 0
	for a, v := range t.Entries {
		block.A[aLen+i] = int32(a)
		block.V[aLen+i] = v
		i++
	}
	block.B[mLen*2] = int16(aLen)
	block.B[mLen*2+1] = int16(aLen + len(t.Entries))
}

func startDate(d []DateRange, elem Date) Date {
	for _, each := range d {
		if each.Start <= elem && each.End >= elem {
			return each.Start
		}
	}
	return 0
}

func startMoment(m []MomentRange, elem Moment) Moment {
	for _, each := range m {
		if each.Start <= elem && each.End >= elem {
			return each.Start
		}
	}
	return 0
}
