package deb

type channelSpace chan *Transaction

func (c channelSpace) Append(s Space) {
	panic("Not implemented")
}

func (c channelSpace) Slice([]Account, []DateRange, []MomentRange) Space {
	panic("Not implemented")
}

func (c channelSpace) Projection([]Account, []DateRange, []MomentRange) Space {
	panic("Not implemented")
}

func (c channelSpace) Transactions() chan *Transaction {
	return c
}
