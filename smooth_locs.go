/*
smooth_locs takes the raw unsmoothed location data and uses an HMM to smooth it.
*/

package main

import (
	"compress/gzip"
	"encoding/gob"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"gonum.org/v1/gonum/floats"

	"github.com/kshedden/rfid/rfid"
)

type personType int

const (
	provider personType = iota
	patient
)

var (
	// The locations
	locs []*rfid.Location

	// The transition probabilities
	trans [][]float64

	// The emission probabilities
	emis [][]float64

	// PIx maps location codes to text location labels.
	IPx map[int]string

	// The input file name
	infname string

	// Extract the field that identifies a distinct person.
	personID personSelector
)

type personSelector func(*rfid.Location) uint64

// readLocs reads all the unsmoothed location records.
func readlocs() []*rfid.Location {

	fid, err := os.Open(infname)
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

	IPcodeRev := make(map[int]string)
	for k, v := range rfid.IPcode {
		IPcodeRev[int(v)] = k
	}

	IPx = make(map[int]string)
	for k, v := range IPcodeRev {
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

// makeTrans constructs the probability transition matrix for the HMM.
func makeTrans(person personType) [][]float64 {

	switch person {
	case patient:
		return makeTransPatient()
	case provider:
		return makeTransProvider()
	default:
		panic("Unkown person type")
	}
}

// makeTransPatient constructsthe probability transition matrix for a patient.
func makeTransPatient() [][]float64 {

	p := len(rfid.IPcode)
	trans := alloc(p, p)

	stick := 50.0

	for j := 0; j < p; j++ {

		exam1 := strings.HasPrefix(IPx[j], "Exam")
		field1 := strings.HasPrefix(IPx[j], "Field")

		for k := 0; k < p; k++ {

			exam2 := strings.HasPrefix(IPx[k], "Exam")
			field2 := strings.HasPrefix(IPx[k], "Field")
			checkin2 := strings.HasPrefix(IPx[k], "Checkin")
			checkoutfinal2 := strings.HasPrefix(IPx[k], "CheckoutFinal")

			switch {
			case j == k:
				trans[j][k] = stick
			case rfid.RoomCode(k) == rfid.Lensometer:
				// Patients can't be in the lensometer room
				trans[j][k] = 0
			case rfid.RoomCode(k) == rfid.Checkout:
				// Can't return to checkout
				trans[j][k] = 0
			case rfid.RoomCode(j) == rfid.CheckoutFinal:
				// Can't leave CheckoutFinal (absorbing state)
				trans[j][k] = 0
			case exam1 && exam2:
				// Patients can't move directly between exam rooms
				trans[j][k] = 0
			case field1 && field2:
				// Patients can't move directly between visual field rooms
				trans[j][k] = 0
			case checkin2:
				// Patient's can't return to checkin
				trans[j][k] = 0
			case checkoutfinal2:
				// Make it easy to go to the absorbing checkout state
				trans[j][k] = stick / 5
			default:
				trans[j][k] = 1
			}
		}
	}

	normalize(trans)

	return trans
}

// makeTransProvider constructs the probability transition matrix for a provider.
func makeTransProvider() [][]float64 {

	p := len(rfid.IPcode)
	trans := alloc(p, p)

	stick := 10.0

	for j := 0; j < p; j++ {
		for k := 0; k < p; k++ {
			switch {
			case j == k:
				trans[j][j] = stick
			default:
				trans[j][k] = 1
			}
		}
	}

	normalize(trans)

	return trans
}

// alloc constructs a m x n rectangular array of float64 arrays.
func alloc(m, n int) [][]float64 {

	mat := make([][]float64, m)
	for j := 0; j < m; j++ {
		mat[j] = make([]float64, n)
	}

	return mat
}

// makeEmission returns an emission probability matrix for each person type.
func makeEmission(person personType) [][]float64 {

	switch person {
	case patient:
		return makeEmissionPatient()
	case provider:
		return makeEmissionProvider()
	default:
		panic("invalid person type\n")
	}
}

// makeExchEmis makes two states in an emission matrix exchangeable.
func makeExchEmis(emis [][]float64, i1, i2 rfid.RoomCode, r float64) [][]float64 {

	emis[i1][i1] = r
	emis[i1][i2] = r
	emis[i2][i1] = r
	emis[i2][i2] = r

	return emis
}

// makeEmissionPatient constucts the emission probability matrix for the HMM for a patient.
func makeEmissionPatient() [][]float64 {

	p := len(IPx)
	emis := alloc(p, p)

	// Ratio of the probability that the observed room is the actual room to the probability that the
	// obvserved room is not the actual room.
	same := 10.0

	for j := 0; j < p; j++ {
		for k := 0; k < p; k++ {

			switch {
			case j == k:
				emis[j][k] = same
			default:
				emis[j][k] = 1
			}
		}
	}

	// Checkout and CheckoutFinal are exchangeable
	emis = makeExchEmis(emis, rfid.Checkout, rfid.CheckoutFinal, same/2)

	// Checkin and NoSignal are exchangeable
	emis = makeExchEmis(emis, rfid.Checkin, rfid.NoSignal, same/2)

	normalize(emis)

	return emis
}

// makeEmissionProvider constucts the emission probability matrix for the HMM for a patient.
func makeEmissionProvider() [][]float64 {

	p := len(IPx)
	emis := alloc(p, p)

	same := 10.0

	for j := 0; j < p; j++ {
		for k := 0; k < p; k++ {
			if j == k {
				emis[j][k] = same
			} else {
				emis[j][k] = 1
			}
		}
	}

	normalize(emis)

	return emis
}

type locsort []*rfid.Location

// Sort location records by person id, tag id, and timestamp.
func (a locsort) Len() int      { return len(a) }
func (a locsort) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a locsort) Less(i, j int) bool {

	if personID(a[i]) < personID(a[j]) {
		return true
	}

	if personID(a[i]) > personID(a[j]) {
		return false
	}
	// below here the ID's are equal

	if a[i].TagId < a[j].TagId {
		return true
	}

	if a[i].TagId > a[j].TagId {
		return false
	}
	// below here the tag id's are equal

	return a[i].TimeStamp.Before(a[j].TimeStamp)
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

// makeStart generates the starting probability distribution for the HMM.
func makeStart(patient bool) []float64 {

	start := make([]float64, len(IPx))
	for i := range start {
		start[i] = 1
	}

	if patient {
		// Rooms where patients cannot go.
		start[rfid.Lensometer] = 0

		// Prefer to start patients at checkin
		start[rfid.Checkin] = 10
	}

	// Normalize
	floats.Scale(1/floats.Sum(start), start)

	return start
}

// process uses the HMM to smooth locations for one person/tag/day.
func process(locs []*rfid.Location) []*rfid.Location {

	locs = continuize(locs)

	hmm := new(rfid.HMM)
	hmm.SetTransmission(trans)
	hmm.SetEmission(emis)

	hmm.SetStart(makeStart(true))

	loci := make([]int, len(locs))
	for i, r := range locs {
		loci[i] = int(r.IP)
	}

	hmm.SetData(loci)
	hmm.Fit()

	for i, r := range locs {
		r.IPhmm = rfid.RoomCode(hmm.Pred[i])
	}

	return locs
}

func run(locs []*rfid.Location) []*rfid.Location {

	var rlocs []*rfid.Location

	i := 0
	for i < len(locs) {

		j := i + 1
		for j < len(locs) {

			if locs[i].TagId != locs[j].TagId || personID(locs[i]) != personID(locs[j]) {
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

// save stores the smoothed locations to a gob file.
func save(locs []*rfid.Location) {

	if !strings.Contains(infname, ".gob.gz") {
		panic("Invalid input file name\n")
	}

	if strings.Contains(infname, "_s.gob.gz") {
		panic("Input data are already smoothed")
	}

	fn := strings.Replace(infname, ".gob.gz", "_s.gob.gz", 1)
	f, err := os.Create(fn)
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

// Use UMid to identify providers
func providerID(r *rfid.Location) uint64 {
	return r.UMid
}

// Use CSN to identify patients
func patientID(r *rfid.Location) uint64 {
	return r.CSN
}

func main() {

	infname = os.Args[1]

	var person personType
	if strings.Contains(strings.ToLower(infname), "provider") {
		person = provider
		personID = providerID
	} else if strings.Contains(strings.ToLower(infname), "patient") {
		person = patient
		personID = patientID
	} else {
		panic("Invalid person type\n")
	}

	setup()

	locs = readlocs()
	sort.Sort(locsort(locs))

	trans = makeTrans(person)
	emis = makeEmission(person)

	rlocs := run(locs)
	save(rlocs)
}
