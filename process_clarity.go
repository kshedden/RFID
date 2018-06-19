/*
Create a gob of sorted Clarity records, sorted by CSN.
*/

package main

import (
	"compress/gzip"
	"encoding/csv"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/transform"

	"github.com/kshedden/rfid/rfid"
)

var (
	recs []*rfid.ClarityRecord
)

// The clarity files use ' instead of " for quoted fields.  Create a
// text.Transformer to replace the single quotes with double quotes.
type fixquote struct{}

func (f *fixquote) Reset() {}

func (f *fixquote) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {

	for i := range src {
		if src[i] == '\'' {
			dst[i] = '"'
		} else {
			dst[i] = src[i]
		}
	}

	if atEOF {
		err = io.EOF
	}

	return len(src), len(src), err
}

func doFile(pa, fname string) {

	fname = path.Join(pa, fname)

	fid, err := os.Open(fname)
	if err != nil {
		panic(err)
	}
	defer fid.Close()

	gid, err := gzip.NewReader(fid)
	if err != nil {
		panic(err)
	}
	defer gid.Close()

	xid := transform.NewReader(gid, &fixquote{})

	rdr := csv.NewReader(xid)
	rdr.FieldsPerRecord = -1

	head, err := rdr.Read()
	if err != nil {
		print(fmt.Sprintf("Can't read header from '%s'\n", fname))
		panic(err)
	}
	cinf := rfid.GetClarityFileInfo(head)

	for {
		rec, err := rdr.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

		for k, v := range rec {
			rec[k] = strings.Replace(v, "'", "", -1)
		}

		cr := new(rfid.ClarityRecord)

		// Parse the CSN
		cr.CSN, err = strconv.ParseUint(rec[cinf.CSN], 10, 64)
		if err != nil {
			panic(err)
		}

		// Parse the check in time
		if len(rec[cinf.CheckInTime]) == 0 {
			continue
		}
		cr.CheckInTime, err = time.Parse("2006-Jan-02 15:04:05", rec[cinf.CheckInTime])
		if err != nil {
			print(rec[cinf.CheckInTime] + "\n")
			panic(err)
		}

		// Parse the check out time
		if len(rec[cinf.CheckOutTime]) == 0 {
			continue
		}
		cr.CheckOutTime, err = time.Parse("2006-Jan-02 15:04:05", rec[cinf.CheckOutTime])
		if err != nil {
			print(rec[cinf.CheckOutTime] + "\n")
			panic(err)
		}

		cr.ProvName = rec[cinf.ProvName]

		cr.VfiOs, err = strconv.ParseFloat(rec[cinf.VfiOs], 64)
		if err != nil {
			cr.VfiOs = math.NaN()
		}

		recs = append(recs, cr)
	}
}

type ByCSN []*rfid.ClarityRecord

func (a ByCSN) Len() int           { return len(a) }
func (a ByCSN) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByCSN) Less(i, j int) bool { return a[i].CSN < a[j].CSN }

func main() {

	pa := path.Join("/", "home", "kshedden", "RFID", "data", "Clarity")
	fnames, err := ioutil.ReadDir(pa)
	if err != nil {
		panic(err)
	}

	for _, finf := range fnames {

		fname := finf.Name()

		if !strings.HasPrefix(fname, "CSN_SUMMARY") {
			continue
		}

		doFile(pa, fname)
	}

	sort.Sort(ByCSN(recs))

	fid, err := os.Create("clarity.gob.gz")
	if err != nil {
		panic(err)
	}
	defer fid.Close()

	gid := gzip.NewWriter(fid)
	defer gid.Close()

	enc := gob.NewEncoder(gid)
	err = enc.Encode(recs)
	if err != nil {
		panic(err)
	}
}
