package main

import (
	"compress/gzip"
	"encoding/gob"
	"io"
	"os"
	"sort"
	"time"

	"github.com/kshedden/rfid/rfid"
)

var (
	// The locations
	locs []*rfid.Location

	// The transition probabilities
	trans [][]float64

	// The emission probabilities
	emis [][]float64

	// Map from location codes to the name of the location
	IPx map[int]string
)

func readlocs() []*rfid.Location {

	fid, err := os.Open("locations.gob.gz")
	if err != nil {
		panic(err)
	}
	defer fid.Close()
	gid, err := gzip.NewReader(fid)
	if err != nil {
		panic(err)
	}

	var locs []*rfid.Location

	dec := gob.NewDecoder(gid)

	for {
		r := new(rfid.Location)
		err := dec.Decode(r)
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		locs = append(locs, r)
	}

	return locs
}

func setup() {

	IPcoder := make(map[int]string)
	for k, v := range rfid.IPcode {
		IPcoder[int(v)] = k
	}

	IPx = make(map[int]string)
	for k, v := range IPcoder {
		IPx[k] = rfid.IPmap[v]
	}
}

func normalize(mat [][]float64) {

	p := len(mat)
	q := len(mat[0])

	for j := 0; j < p; j++ {
		f := 0.0
		for k := 0; k < q; k++ {
			f += mat[j][k]
		}
		for k := 0; k < q; k++ {
			mat[j][k] /= f
		}
	}
}

func makeTrans() [][]float64 {

	p := len(rfid.IPcode)
	trans := alloc(p, p)

	for j := 0; j < p; j++ {

		//		exam1 := strings.HasPrefix(IPx[j], "Exam")

		for k := 0; k < p; k++ {

			//			exam2 := strings.HasPrefix(IPx[k], "Exam")

			switch {
			case j == k:
				trans[j][k] = 100

			default:
				trans[j][k] = 1
			}
		}
	}

	// Can never go back to checkout
	for j := 0; j < p; j++ {
		if j != int(rfid.Checkout) {
			trans[j][rfid.Checkout] = 0
		}
	}

	// Can never leave CheckoutReturn
	for j := 0; j < p; j++ {
		if j != int(rfid.CheckoutReturn) {
			trans[rfid.CheckoutReturn][j] = 0
		}
	}

	// Make it easier to go to CheckoutReturn
	for j := 0; j < p; j++ {
		// TODO needs adjusting
		trans[j][rfid.CheckoutReturn] = 20
	}

	normalize(trans)

	return trans
}

func alloc(m, n int) [][]float64 {

	mat := make([][]float64, m)
	for j := 0; j < m; j++ {
		mat[j] = make([]float64, n)
	}

	return mat
}

func makeEmission() [][]float64 {

	p := len(IPx)
	emis := alloc(p, p)

	for j := 0; j < p; j++ {
		for k := 0; k < p; k++ {

			switch {
			case j == k:
				emis[j][k] = 5
			default:
				emis[j][k] = 1
			}
		}
	}

	// Checkout and CheckoutReturn are exchangeable
	emis[rfid.Checkout][rfid.Checkout] = 2.5
	emis[rfid.Checkout][rfid.CheckoutReturn] = 2.5
	emis[rfid.CheckoutReturn][rfid.Checkout] = 2.5
	emis[rfid.Checkout][rfid.Checkout] = 2.5

	normalize(emis)

	return emis
}

type Locx []*rfid.Location

func (a Locx) Len() int      { return len(a) }
func (a Locx) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a Locx) Less(i, j int) bool {

	if a[i].TagId < a[j].TagId {
		return true
	}

	if a[i].TagId > a[j].TagId {
		return false
	}
	// below here the tag id's are equal

	if a[i].CSN < a[j].CSN {
		return true
	}

	if a[i].CSN > a[j].CSN {
		return false
	}
	// below here the CSN's are equal

	return a[i].TimeStamp.Before(a[j].TimeStamp)
}

func argmax(x []float64) int {

	i := 0
	v := x[0]

	for j := 1; j < len(x); j++ {
		if x[j] > v {
			i = j
			v = x[j]
		}
	}

	return i
}

// continuize fills in the gaps where no signal was detected for a tag
func continuize(locs []*rfid.Location) []*rfid.Location {

	var locx []*rfid.Location
	locx = append(locx, locs[0])

	for i := 1; i < len(locs); i++ {

		lastloc := locx[len(locx)-1]

		for locs[i].TimeStamp.Sub(lastloc.TimeStamp).Minutes() > 1 {
			x := new(rfid.Location)
			*x = *lastloc
			x.TimeStamp = x.TimeStamp.Add(time.Minute)
			x.IP = rfid.NoSignal
			x.Signal = 0
			x.IP2 = rfid.Null
			x.Signal2 = 0
			lastloc = x
			locx = append(locx, x)
		}

		locx = append(locx, locs[i])
	}

	return locx
}

func process(locs []*rfid.Location) []*rfid.Location {

	locs = continuize(locs)

	hmm := new(rfid.HMM)
	hmm.SetTransmission(trans)
	hmm.SetEmission(emis)

	start := make([]float64, len(IPx))
	u := 1 / float64(len(IPx))
	for i := range start {
		start[i] = u
	}
	hmm.SetStart(start)

	loci := make([]int, len(locs))
	for i, r := range locs {
		loci[i] = int(r.IP)
	}

	hmm.SetData(loci)
	hmm.Fit()

	for i, r := range locs {
		r.IPhmm = rfid.RoomCode(argmax(hmm.PostProb[i]))
	}

	return locs
}

func run(locs []*rfid.Location) []*rfid.Location {

	var rlocs []*rfid.Location

	i := 0
	for i < len(locs) {

		j := i + 1
		for j < len(locs) {

			if locs[i].TagId != locs[j].TagId || locs[i].CSN != locs[j].CSN {
				break
			}

			if !locs[i].TimeStamp.Truncate(24 * time.Hour).Equal(locs[j].TimeStamp.Truncate(24 * time.Hour)) {
				break
			}

			j++
		}

		rlocs = append(rlocs, process(locs[i:j])...)
		i = j
	}

	return rlocs
}

func save(locs []*rfid.Location) {

	f, err := os.Create("locations_s.gob.gz")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	z := gzip.NewWriter(f)
	defer z.Close()

	enc := gob.NewEncoder(z)

	for _, r := range locs {
		err := enc.Encode(&r)
		if err != nil {
			panic(err)
		}
	}
}

func main() {

	setup()

	locs = readlocs()
	sort.Sort(Locx(locs))

	trans = makeTrans()
	emis = makeEmission()

	rlocs := run(locs)
	save(rlocs)
}
