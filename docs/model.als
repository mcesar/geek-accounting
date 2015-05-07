module deb

sig A {}
sig D {}
sig M {}

sig Db {} // Debit
sig Cr {} // Credit

sig S {
	_: M -> D -> A -> set (Db + Cr)
}
{	
	all m: M, d: D | #_[m,d].(Cr+Db) > 1
	all m: M, d: D, a: A | _[m,d,a] in Db or _[m,d,a] in Cr
	all m: M, d: D | #_[m,d,A] & Db = #_[m,d,A] & Cr
	//no disj a1, a2 : A, x: (Db+Cr) | x in _[M,D,a1] and x in _[M,D,a2]
	all x : (Db+Cr) | lone x[_] 
}

pred show(s : S) {
	#s._[M,D].(Db+Cr) > 2
}

run show for 6 but 1 S, 3 A
