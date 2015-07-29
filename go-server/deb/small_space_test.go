package deb

import "testing"

type SmallSpaceBuilder int

func (SmallSpaceBuilder) NewSpace(arr Array) Space {
	return NewSmallSpace(arr, nil)
}

func (SmallSpaceBuilder) NewSpaceWithOffset(arr Array, do, mo int) Space {
	return NewSmallSpaceWithOffset(arr, do, mo, nil)
}

func TestSmallSpaceTransactions(t *testing.T) {
	SpaceTester(0).TestTransactions(t, SmallSpaceBuilder(0))
}
func TestSmallSpaceAppend(t *testing.T) {
	SpaceTester(0).TestAppend(t, SmallSpaceBuilder(0))
}
func TestSmallSpaceSlice(t *testing.T) {
	SpaceTester(0).TestSlice(t, SmallSpaceBuilder(0))
}
func TestSmallSpaceProjection(t *testing.T) {
	SpaceTester(0).TestProjection(t, SmallSpaceBuilder(0))
}
