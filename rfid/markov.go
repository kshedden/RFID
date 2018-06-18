package rfid

import (
	"fmt"

	"gonum.org/v1/gonum/floats"
)

// HMM represents a Hidden Markov Model.
type HMM struct {

	// The data sequence, coded 0, 1, ...
	data []int

	// The emission probabilities
	emis [][]float64

	// The marginal distribution of the first point
	start []float64

	// The transition probabilities
	trans [][]float64

	// The forward probabilites
	fprob [][]float64

	// The backward probabilites
	bprob [][]float64

	// Posterior probabilities
	PostProb [][]float64

	// Number of time points
	nTime int

	// Number of states
	nState int
}

// alloc creates a rectangular float64 array of arrays.
func alloc(row, col int) [][]float64 {

	x := make([][]float64, row)
	for i := 0; i < row; i++ {
		x[i] = make([]float64, col)
	}

	return x
}

func probnorm(probs []float64) {
	mx := floats.Max(probs)
	floats.Scale(1/mx, probs)
}

// forward calculates the forward HMM probabilities.
func (hmm *HMM) forward() {

	for j := 0; j < hmm.nState; j++ {
		hmm.fprob[0][j] = hmm.start[j] * hmm.emis[j][hmm.data[0]]
	}

	for t := 1; t < hmm.nTime; t++ {
		for j := 0; j < hmm.nState; j++ {
			f := hmm.emis[j][hmm.data[t]]
			for k := 0; k < hmm.nState; k++ {
				hmm.fprob[t][j] += f * hmm.fprob[t-1][k] * hmm.trans[k][j]
			}
		}

		probnorm(hmm.fprob[t])
	}
}

// backward calculates the backward HMM probabilities.
func (hmm *HMM) backward() {

	for j := 0; j < hmm.nState; j++ {
		hmm.bprob[hmm.nTime-1][j] = 1
	}

	for t := hmm.nTime - 2; t >= 0; t-- {
		for j := 0; j < hmm.nState; j++ {
			for k := 0; k < hmm.nState; k++ {
				hmm.bprob[t][j] += hmm.bprob[t+1][k] * hmm.trans[j][k] * hmm.emis[k][hmm.data[t+1]]
			}
		}

		probnorm(hmm.bprob[t])
	}
}

// getPost calculates the conditional probabilities of the state sequence
// given the data.
func (hmm *HMM) getPost() {

	hmm.PostProb = alloc(hmm.nTime, hmm.nState)

	for t := 0; t < hmm.nTime; t++ {
		d := 0.0
		for j := 0; j < hmm.nState; j++ {
			d += hmm.fprob[t][j] * hmm.bprob[t][j]
		}
		if d == 0 {
			fmt.Printf("%v\n", hmm.fprob[t])
			fmt.Printf("%v\n", hmm.bprob[t])
			panic("ZERO\n")
		}
		for j := 0; j < hmm.nState; j++ {
			hmm.PostProb[t][j] = hmm.fprob[t][j] * hmm.bprob[t][j] / d
		}
	}
}

// SetEmission sets tbe emission probabilities.
func (hmm *HMM) SetEmission(emis [][]float64) {
	hmm.emis = emis
}

// SetTransmission sets the transmission probability matrix.
func (hmm *HMM) SetTransmission(trans [][]float64) {
	hmm.trans = trans
}

// SetData sets the data to which the HMM will be fit.
func (hmm *HMM) SetData(data []int) {
	hmm.data = data
}

// SetStart sets the starting probability distribution.
func (hmm *HMM) SetStart(start []float64) {
	hmm.start = start
}

// Fit calculates the posterior probabilities of the HMM based on the data.
func (hmm *HMM) Fit() {

	hmm.nTime = len(hmm.data)
	hmm.nState = len(hmm.emis)

	hmm.fprob = alloc(hmm.nTime, hmm.nState)
	hmm.bprob = alloc(hmm.nTime, hmm.nState)

	hmm.forward()
	hmm.backward()
	hmm.getPost()
}
