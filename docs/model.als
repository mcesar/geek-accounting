module model
open util/ordering[D]
open util/ordering[M]

sig A {}
sig D {}
sig M {}

sig Debit {}
sig Credit {}
sig V in Debit+Credit {}

sig DI { 
	start: D,
	end: D } {
	end in start.next
	}
sig MI {
	start: M,
	end: M} {
	end in start.next
	}

sig S { arr: MI -> DI -> A -> V } {	
	all m: MI, d: DI | #arr[m,d].V > 1
	all m: MI, d: DI, a: A | arr[m,d,a] in Debit or arr[m,d,a] in Credit
	all m: MI, d: DI | #arr[m,d,A] & Debit = #arr[m,d,A] & Credit
	}

fact DebitAndCreditAreUnitaryUnitsOfAccount {
	all x : V | lone arr.x
	}

assert theSumOfDebitsMustBeEqualsTheSumOfCreditsForTheWholeSpace {
	all s: S | #s.arr[MI,DI,A] & Debit = #s.arr[MI,DI,A] & Credit
}
check theSumOfDebitsMustBeEqualsTheSumOfCreditsForTheWholeSpace
	for 5 but 1 S, 3 A, 5 Int

pred show(s : S) {
	some m: MI, d: DI | 
		#s.arr[m,d].V > 2 and  
		#s.arr.V.A[m] > 1 and 
		#s.arr.V.A.DI > 1
	}

run show for 5 but 1 S, 3 A, 5 Int
