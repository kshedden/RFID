package rfid

import (
	"math"
	"testing"
)

/*
library(HMM)

# Test case 1
tp = matrix(c(0.6, 0.4, 0.3, 0.7), c(2, 2), byrow=T)
sp = c(0.5, 0.5)
ep = matrix(c(0.3, 0.4, 0.3, 0.4, 0.3, 0.3), c(2, 3), byrow=T)
hmm = initHMM(c("S", "T"), c("A","B", "C"), transProbs=tp,
              startProbs=sp, emissionProbs=ep)
obs = c("A", "B", "C", "C")
post = posterior(hmm, obs)
vit = viterbi(hmm, obs)

# Test case 2
tp = matrix(c(0.8, 0.2, 0.1, 0.9), c(2, 2), byrow=T)
sp = c(0.5, 0.5)
ep = matrix(c(0.9, 0.1, 0.2, 0.8), c(2, 2), byrow=T)
hmm = initHMM(c("S", "T"), c("A","B"), transProbs=tp,
              startProbs=sp, emissionProbs=ep)
obs = c("A", "A", "B", "B")
post = posterior(hmm, obs)
vit = viterbi(hmm, obs)
*/

// Comparisons to R HMM package.
func TestMarkov(t *testing.T) {

	for _, q := range []struct {
		emis  [][]float64
		trans [][]float64
		start []float64
		data  []int
		post  [][]float64
		vit   []int
	}{
		{
			emis:  [][]float64{[]float64{0.3, 0.4, 0.3}, []float64{0.4, 0.3, 0.3}},
			trans: [][]float64{[]float64{0.6, 0.4}, []float64{0.3, 0.7}},
			start: []float64{0.5, 0.5},
			data:  []int{0, 1, 2, 2},
			post:  [][]float64{[]float64{0.45, 0.55}, []float64{0.5, 0.5}, []float64{0.45, 0.55}, []float64{0.435, 0.565}},
			vit:   []int{1, 1, 1, 1},
		},
		{
			emis:  [][]float64{[]float64{0.9, 0.1}, []float64{0.2, 0.8}},
			trans: [][]float64{[]float64{0.8, 0.2}, []float64{0.1, 0.9}},
			start: []float64{0.8, 0.2},
			data:  []int{0, 0, 1, 1},
			post: [][]float64{[]float64{0.9523, 0.0477}, []float64{0.7888, 0.2112}, []float64{0.1123, 0.8877},
				[]float64{0.0496, 0.9504}},
			vit: []int{0, 0, 1, 1},
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
					t.Fail()
				}
			}
		}

		for i := 0; i < len(q.data); i++ {
			if hmm.Pred[i] != q.vit[i] {
				t.Fail()
			}
		}
	}
}
