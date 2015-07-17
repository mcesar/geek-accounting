package deb

type Account uint32
type Date uint32
type Moment uint64
type DateRange struct{ Start, End Date }
type MomentRange struct{ Start, End Moment }
type Entries map[Account]int64

type Transaction struct {
	Moment  Moment
	Date    Date
	Entries Entries
}

type Space interface {
	Append(Space)
	Slice([]Account, []DateRange, []MomentRange) Space
	Projection([]Account, []DateRange, []MomentRange) Space
	Transactions() chan *Transaction
}
