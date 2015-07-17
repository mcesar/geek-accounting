package deb

import "fmt"

type largeSpace struct {
	blockSize  uint
	dataBlocks []*dataBlock
}

type dataBlock struct {
	m []uint64
	d []uint32
	a []uint32
	v []int64
	b []uint16
}

func NewLargeSpace(blockSize uint) *largeSpace {
	ls := largeSpace{blockSize: blockSize}
	return &ls
}

func (ls *largeSpace) Append(s Space) {
	for t := range s.Transactions() {
		db := ls.lastDataBlock()
		if db == nil || uint(len(db.a)+len(t.Entries)) > ls.memberSize() {
			db = ls.newDataBlock()
			if ls.dataBlocks == nil {
				ls.dataBlocks = []*dataBlock{db}
			} else {
				ls.dataBlocks = append(ls.dataBlocks, db)
			}
		}
		db.append(t)
	}
}

func (ls *largeSpace) Slice(a []Account, d []DateRange, m []MomentRange) Space {
	out := make(chan *Transaction)
	go func() {
		defer close(out)
		ls.iterateWithFilter(a, d, m, func(db *dataBlock, i int) { out <- db.newTransaction(i) })
	}()
	return channelSpace(out)
}

func (ls *largeSpace) Projection(a []Account, d []DateRange, m []MomentRange) Space {
	type key struct {
		moment Moment
		date   Date
	}
	transactions := map[key]*Transaction{}
	ls.iterateWithFilter(a, d, m, func(db *dataBlock, i int) {
		k := key{startMoment(m, Moment(db.m[i])), startDate(d, Date(db.d[i]))}
		nt := db.newTransaction(i)
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
	out := make(chan *Transaction)
	go func() {
		defer close(out)
		for _, t := range transactions {
			out <- t
		}
	}()
	return channelSpace(out)
}

func (ls *largeSpace) Transactions() chan *Transaction {
	out := make(chan *Transaction)
	go func() {
		defer close(out)
		for _, db := range ls.dataBlocks {
			for i := 0; i < len(db.m); i++ {
				out <- db.newTransaction(i)
			}
		}
	}()
	return out
}

func (ls *largeSpace) String() string {
	blocksAsString := make([]string, len(ls.dataBlocks))
	for i, db := range ls.dataBlocks {
		blocksAsString[i] = fmt.Sprintf("%v", *db)
	}
	return fmt.Sprintf("{%v %v %v %v}",
		ls.blockSize, len(ls.dataBlocks), ls.memberSize(), blocksAsString)
}

func (db *dataBlock) newTransaction(i int) *Transaction {
	t := Transaction{Moment(db.m[i]), Date(db.d[i]), make(Entries)}
	for j := db.b[i*2]; j < db.b[i*2+1]; j++ {
		t.Entries[Account(db.a[j])] = db.v[j]
	}
	return &t
}

func (ls *largeSpace) lastDataBlock() *dataBlock {
	if len(ls.dataBlocks) == 0 {
		return nil
	}
	return ls.dataBlocks[len(ls.dataBlocks)-1]
}

func (ls *largeSpace) memberSize() uint {
	return ls.blockSize / (64 + 32 + 32*2 + 64*2 + 16*2)
}

func (ls *largeSpace) newDataBlock() *dataBlock {
	db := new(dataBlock)
	db.m = make([]uint64, 0, ls.memberSize())
	db.d = make([]uint32, 0, ls.memberSize())
	db.a = make([]uint32, 0, ls.memberSize()*2)
	db.v = make([]int64, 0, ls.memberSize()*2)
	db.b = make([]uint16, 0, ls.memberSize()*2)
	return db
}

func (ls *largeSpace) iterateWithFilter(a []Account, d []DateRange, m []MomentRange,
	f func(*dataBlock, int)) {
	for _, db := range ls.dataBlocks {
		for i := 0; i < len(db.m); i++ {
			if containsMoment(m, Moment(db.m[i])) && containsDate(d, Date(db.d[i])) {
				for j := db.b[i*2]; j < db.b[i*2+1]; j++ {
					if containsAccount(a, Account(db.a[j])) {
						f(db, i)
						break
					}
				}
			}
		}
	}
}

func (db *dataBlock) append(t *Transaction) {
	mLen := len(db.m)
	aLen := len(db.a)
	db.m = db.m[0 : mLen+1]
	db.d = db.d[0 : mLen+1]
	db.a = db.a[0 : aLen+len(t.Entries)]
	db.v = db.v[0 : aLen+len(t.Entries)]
	db.b = db.b[0 : mLen*2+2]
	db.m[mLen] = uint64(t.Moment)
	db.d[mLen] = uint32(t.Date)
	i := 0
	for a, v := range t.Entries {
		db.a[aLen+i] = uint32(a)
		db.v[aLen+i] = v
		i++
	}
	db.b[mLen*2] = uint16(aLen)
	db.b[mLen*2+1] = uint16(aLen + len(t.Entries))
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
