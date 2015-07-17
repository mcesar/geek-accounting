package deb

import "testing"

type LargeSpaceBuilder int

func (lsb LargeSpaceBuilder) NewSpace(arr Array) Space {
	return lsb.NewSpaceWithOffset(arr, 0, 0)
}

func (LargeSpaceBuilder) NewSpaceWithOffset(arr Array, do, mo int) Space {
	ls := NewLargeSpace(10 * 1024)
	ls.Append(NewSmallSpaceWithOffset(arr, do, mo))
	return ls
}

func TestLargeSpaceTransactions(t *testing.T) {
	SpaceTester(0).TestTransactions(t, LargeSpaceBuilder(0))
}
func TestLargeSpaceAppend(t *testing.T) {
	SpaceTester(0).TestAppend(t, LargeSpaceBuilder(0))
}
func TestLargeSpaceSlice(t *testing.T) {
	SpaceTester(0).TestSlice(t, LargeSpaceBuilder(0))
}
func TestLargeSpaceProjection(t *testing.T) {
	SpaceTester(0).TestProjection(t, LargeSpaceBuilder(0))
}
