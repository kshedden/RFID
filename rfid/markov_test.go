package rfid

import (
	"math"
	"testing"
)

// Comparisons to R HMM package.
func TestMarkov(t *testing.T) {

	for _, q := range []struct {
		emis  [][]float64
		trans [][]float64
		start []float64
		data  []int
		post  [][]float64
	}{
		{
			emis:  [][]float64{[]float64{0.3, 0.4, 0.3}, []float64{0.4, 0.3, 0.3}},
			trans: [][]float64{[]float64{0.6, 0.4}, []float64{0.3, 0.7}},
			start: []float64{0.5, 0.5},
			data:  []int{0, 1, 2, 2},
			post:  [][]float64{[]float64{0.45, 0.55}, []float64{0.5, 0.5}, []float64{0.45, 0.55}, []float64{0.435, 0.565}},
		},
		{
			emis:  [][]float64{[]float64{0.9, 0.1}, []float64{0.2, 0.8}},
			trans: [][]float64{[]float64{0.8, 0.2}, []float64{0.1, 0.9}},
			start: []float64{0.8, 0.2},
			data:  []int{0, 0, 1, 1},
			post: [][]float64{[]float64{0.9523, 0.0477}, []float64{0.7888, 0.2112}, []float64{0.1123, 0.8877},
				[]float64{0.0496, 0.9504}},
		},
	} {

		hmm := HMM{}
		hmm.SetEmission(q.emis)
		hmm.SetTransmission(q.trans)
		hmm.SetData(q.data)
		hmm.SetStart(q.start)
		hmm.Fit()

		for i := 0; i < len(q.data); i++ {
			for j := 0; j < len(q.start); j++ {
				if math.Abs(q.post[i][j]-hmm.PostProb[i][j]) > 1e-4 {
					print(i, " ", j, "\n")
					t.Fail()
				}
			}
		}
	}
}
