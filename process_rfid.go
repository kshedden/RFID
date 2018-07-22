package main

import (
	"compress/gzip"
	"encoding/csv"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"sort"
	"time"

	"github.com/kshedden/rfid/rfid"
)

var (

	// All the Clarity records
	clarity []*rfid.ClarityRecord

	logger *log.Logger
)

type byTime []*rfid.RFIDrecord

func (a byTime) Len() int           { return len(a) }
func (a byTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byTime) Less(i, j int) bool { return a[i].TimeStamp.Before(a[j].TimeStamp) }

func readDay(year, month, day int) (*rfid.RFIDinfo, []*rfid.RFIDrecord, []*rfid.RFIDrecord) {

	fname := fmt.Sprintf("%4d-%02d-%02d_APD.csv.gz", year, month, day)
	fname = path.Join("/", "home", "kshedden", "RFID", "data", "APD", fname)

	// If the file does not exist, return silently
	if _, err := os.Stat(fname); err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
	}
	logger.Print(fmt.Sprintf("Processing file '%s'", fname))

	fid, err := os.Open(fname)
	if err != nil {
		panic(err)
	}
	defer fid.Close()
	gid, err := gzip.NewReader(fid)
	if err != nil {
		panic(err)
	}

	rdr := csv.NewReader(gid)
	rdr.ReuseRecord = true

	var patrecs, provrecs []*rfid.RFIDrecord
	var n int
	var rfi rfid.RFIDinfo
	var nerr int
	for {
		fields, err := rdr.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			nerr++
			continue
		}

		n++

		r := new(rfid.RFIDrecord)
		if !r.Parse(fields, &rfi) {
			continue
		}

		// Exclude records when clinic is closed
		if r.TimeStamp.Hour() < 7 {
			rfi.TimeEarly++
			continue
		}
		if r.TimeStamp.Hour() > 19 {
			rfi.TimeLate++
			continue
		}

		switch r.PersonCat {

		case rfid.Patient:

			// Check if the CSN is in the Clarity data
			ii := sort.Search(len(clarity), func(i int) bool { return r.CSN <= clarity[i].CSN })
			if ii == len(clarity) || clarity[ii].CSN != r.CSN {
				rfi.NoClarity++
				continue
			}
			for j := ii; clarity[j].CSN == r.CSN; j++ {
				// We found a CSN match, but also need to check the date.
				if clarity[j].CheckInTime.Truncate(24*time.Hour) == r.TimeStamp.Truncate(24*time.Hour) {
					r.Clarity = clarity[j]
				}
			}
			if r.Clarity == nil {
				rfi.NoClarity++
				continue
			}

			// Check if the time is prior to the Clarity check-in time
			if r.TimeStamp.Before(clarity[ii].CheckInTime) {
				rfi.BeforeCheckIn++
				continue
			}

			// Check if the time is after the Clarity check-out time
			if r.TimeStamp.After(clarity[ii].CheckOutTime) {
				rfi.AfterCheckOut++
				continue
			}

			// Keep a reference to the Clarity record
			r.Clarity = clarity[ii]

			patrecs = append(patrecs, r)

		case rfid.Provider:
			provrecs = append(provrecs, r)

		default:
			panic("Unkown person type\n")
		}
	}

	if nerr > 0 {
		print("Errors parsing CSV file, see log for more information\n")
	}
	logger.Printf("%d errors parsing csv file", nerr)

	// Confirm that it is sorted by time
	sort.Sort(byTime(provrecs))
	sort.Sort(byTime(patrecs))

	provrecs = spantime(provrecs, &rfi)
	patrecs = spantime(patrecs, &rfi)

	rfi.FileName = fname
	rfi.TotalRecs = n
	rfi.FinalRecs = len(provrecs) + len(patrecs)

	return &rfi, patrecs, provrecs
}

// spantime removes records from a given IP source if there have
// already been two records from the same source in the last second.
func spantime(recs []*rfid.RFIDrecord, rfi *rfid.RFIDinfo) []*rfid.RFIDrecord {

	last1 := make([]time.Time, 256)
	last2 := make([]time.Time, 256)

	for k, r := range recs {

		// Initial fill-in of the queues
		if last1[r.IP].IsZero() {
			last1[r.IP] = r.TimeStamp
			continue
		} else if last2[r.IP].IsZero() {
			last2[r.IP] = last1[r.IP]
			last1[r.IP] = r.TimeStamp
			continue
		}

		if r.TimeStamp.Sub(last2[r.IP]).Seconds() <= 1 {
			rfi.TimeSpanFull++
			recs[k] = nil
		}

		// Shift down
		last2[r.IP] = last1[r.IP]
		last1[r.IP] = r.TimeStamp
	}

	// Remove the nil values.
	recx := make([]*rfid.RFIDrecord, len(recs))
	var i int
	for _, v := range recs {
		if v != nil {
			recx[i] = v
			i++
		}
	}
	recx = recx[0:i]

	return recx
}

func readClarity() {

	fid, err := os.Open("clarity.gob.gz")
	if err != nil {
		panic(err)
	}
	defer fid.Close()

	gid, err := gzip.NewReader(fid)
	if err != nil {
		panic(err)
	}
	defer gid.Close()

	dec := gob.NewDecoder(gid)

	dec.Decode(&clarity)
}

func setupLog() {
	fid, err := os.Create("process_rfid.log")
	if err != nil {
		panic(err)
	}
	logger = log.New(fid, "", 0)
}

func main() {

	setupLog()

	readClarity()

	// Setup encoders for patients and providers
	var enc [2]*gob.Encoder
	for j := 0; j < 2; j++ {
		fname := "patient_locations.gob.gz"
		if j == 1 {
			fname = "provider_locations.gob.gz"
		}
		f, err := os.Create(fname)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		g := gzip.NewWriter(f)
		defer g.Close()

		enc[j] = gob.NewEncoder(g)
	}

	for year := 2018; year <= 2018; year++ {
		for month := 1; month <= 12; month++ {
			for day := 1; day <= 31; day++ {

				rif, patrecs, provrecs := readDay(year, month, day)
				fmt.Printf("%d-%d-%d %d %d\n", year, month, day, len(provrecs), len(patrecs))

				// Should do something with this
				_ = rif

				patlocs := rfid.GetLocation(patrecs)
				provlocs := rfid.GetLocation(provrecs)

				for _, loc := range patlocs {
					err := enc[0].Encode(loc)
					if err != nil {
						panic(err)
					}
				}

				for _, loc := range provlocs {
					err := enc[1].Encode(loc)
					if err != nil {
						panic(err)
					}
				}
			}
		}
	}
}
