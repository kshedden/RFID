/*
match_locs assesses for each patient minute whether a provider is present,
and for each provider whether a patient is present.
*/

package main

import (
	"compress/gzip"
	"encoding/gob"
	"io"
	"os"
	"sort"

	"github.com/kshedden/rfid/rfid"
)

var (
	// Location records for providers
	providers []*rfid.Location

	// Location records for patients
	patients []*rfid.Location
)

// Enable sorting of locations by time
type byTime []*rfid.Location

func (a byTime) Len() int           { return len(a) }
func (a byTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byTime) Less(i, j int) bool { return a[i].TimeStamp.Before(a[j].TimeStamp) }

// Enable sorting of locations by CSN
type byCSN []*rfid.Location

func (a byCSN) Len() int           { return len(a) }
func (a byCSN) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byCSN) Less(i, j int) bool { return a[i].CSN < a[j].CSN }

// load reads an array of location records from a gzipped gob file.
func load(fname string) []*rfid.Location {

	f, err := os.Open(fname)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	g, err := gzip.NewReader(f)
	if err != nil {
		panic(err)
	}
	defer g.Close()

	dec := gob.NewDecoder(g)

	var rec []*rfid.Location

	for {
		r := new(rfid.Location)
		err := dec.Decode(r)
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		rec = append(rec, r)
	}

	return rec
}

// save writes an array of location records to a gzipped gob file.
func save(fname string, recs []*rfid.Location) {

	f, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	g := gzip.NewWriter(f)
	defer g.Close()

	enc := gob.NewEncoder(g)

	for _, r := range recs {
		err := enc.Encode(r)
		if err != nil {
			panic(err)
		}
	}
}

func search() {

	for _, pat := range patients {

		if pat.IPhmm == rfid.NoSignal {
			continue
		}

		f := func(j int) bool {
			return !providers[j].TimeStamp.Before(pat.TimeStamp)
		}

		ii := sort.Search(len(providers), f)

		if !providers[ii].TimeStamp.Equal(pat.TimeStamp) {
			continue
		}

		for k := ii; k < len(providers) && providers[k].TimeStamp.Equal(pat.TimeStamp); k++ {
			if pat.IPhmm == providers[k].IPhmm {
				pat.Match = true
				providers[k].Match = true
				break
			}
		}
	}
}

func main() {

	patients = load("patient_locations_s.gob.gz")
	providers = load("provider_locations_s.gob.gz")

	sort.Sort(byTime(providers))
	sort.Sort(byCSN(patients))

	search()

	save("patient_locations_sm.gob.gz", patients)
	save("provider_locations_sm.gob.gz", providers)
}
