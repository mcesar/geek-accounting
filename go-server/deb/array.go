package deb

type Array [][][]int64

func (arr Array) Copy() (result Array) {
	x, y, z := arr.Dimensions()
	if arr.Empty() {
		return Array{{{}}}
	}
	result = make(Array, x)
	values := make([]int64, x*y*z)
	for i := range result {
		result[i] = make([][]int64, y)
		for j := range result[i] {
			result[i][j], values = values[:z], values[z:]
			copy(result[i][j], arr[i][j])
		}
	}
	return
}

func (arr Array) Transposed() (result Array) {
	x, y, z := arr.Dimensions()
	if arr.Empty() {
		return Array{{{}}}
	}
	result = make(Array, z)
	values := make([]int64, x*y*z)
	for i := range result {
		result[i] = make([][]int64, y)
		for j := range result[i] {
			result[i][j], values = values[:x], values[x:]
			for k := range result[i][j] {
				result[i][j][k] = arr[k][j][i]
			}
		}
	}
	return
}

func (arr Array) Dimensions() (int, int, int) {
	if len(arr) == 0 || len(arr[0]) == 0 || len(arr[0][0]) == 0 {
		return 0, 0, 0
	} else {
		return len(arr), len(arr[0]), len(arr[0][0])
	}
}

func (arr Array) Empty() bool {
	x, y, z := arr.Dimensions()
	return x == 0 || y == 0 || z == 0
}
