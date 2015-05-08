module model2
open util/ordering[D] as do
open util/ordering[M] as mo
open util/integer

sig A {}
sig D {}
sig M {}

sig DI { start: D, end: D } { start.lte[end] }
sig MI { start: M, end: M } { start.lte[end] }

fact {
	no disj di, di': DI | di.start = di'.start and di.end = di'.end
	no disj mi, mi': MI | mi.start = mi'.start and mi.end = mi'.end
	}

sig Index { 
	i, j, k: Int } 
	{ 
		i > 0 and j > 0 and k > 0
	}

fact {
	no disj idx, idx': Index | idx.i = idx'.i and idx.j = idx'.j and idx.k = idx'.k
	}

sig S {
	a: seq A,
	d: seq DI,
	m: seq MI,
	arr: Index -> lone Int} 
	{
	not (a.hasDups or d.hasDups or m.hasDups)
	all idx: arr.univ | idx.i <= #a and idx.j <= #d and idx.k <= #m 
	no 0 & univ.arr
	all j: d.inds, k: m.inds | 	
		(sum idx : (arr.univ).filter[j.plus[1], k.plus[1]] | arr[idx]) = 0
	}

fact sumOfTheWholeSpaceIsZero {
	all s: S | (sum idx: s.arr.univ | s.arr[idx]) = 0
	}

pred append (disj s, s', s'': S) {
	mo/max[s.m.elems.end].lt[mo/min[s'.m.elems.start]]
	s''.m = s.m.append[s'.m]
	s''.d = s.d.append[seq (s'.d.elems-s.d.elems)]
	}

fun filter (indexes: set Index, j', k': Int): set Index {
	{ idx: indexes | idx.j = j' and idx.k = k' }
	}

pred small(s: S) {
	#s.a <= 5
	#s.d <= 5
	#s.m <= 5
	all v: s.arr[univ] | v >= -5 and v <= 5 
	}

pred interesting(s: S) {
	some s.arr
	}

pred veryInteresting(s: S) {
	#s.arr.univ.j > 1
	#s.arr.univ.k > 1
	some disj idx, idx': s.arr.univ | idx.j != idx'.j and idx.k = idx'.k
	some disj idx, idx', idx'': s.arr.univ | 
		idx.j = idx'.j and idx.j = idx''.j and idx.k = idx'.k and idx.k = idx''.k
	}

pred allSmallAndInteresting {
	some S
	all s: S | s.small and s.interesting
	some s: S | s.small and s.veryInteresting
	}

pred showAppend(disj s, s', s'': S) {
	allSmallAndInteresting
	append[s, s', s'']
	}

run allSmallAndInteresting for 7 but 3 A, 3 D, 3 M, 3 S, 6 Int
run showAppend for 7 but 3 A, 3 D, 3 M, 3 S, 6 Int
