module deb

sig A {}
sig D {}
sig M {}
sig S {
	_: M -> D -> A -> lone Int
}
{	
	all m: M, d: D | #d.(m._) > 1
	all m: M, d: D, a: A | a.(d.(m._)) != 0
	all m: M, d: D | sum A.(d.(m._)) = 0
}

pred show(s : S) {
	some s._
}
run show for 3 but 1 S
