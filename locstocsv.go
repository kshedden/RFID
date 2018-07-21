/*
Convert the locations gob file to csv format.
*/

package main

import (
	"compress/gzip"
	"encoding/csv"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kshedden/rfid/rfid"
)

func main() {

	fname := os.Args[1]

	if !strings.HasSuffix(fname, ".gob.gz") {
		panic("file name must end in '.gob.gz'")
	}

	f, err := os.Open(fname)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		panic(err)
	}
	defer z.Close()
	dec := gob.NewDecoder(z)

	outn := strings.Replace(fname, ".gob.gz", ".csv.gz", -1)
	if outn == fname {
		panic("Wrong file type")
	}
	outf, err := os.Create(outn)
	if err != nil {
		panic(err)
	}
	defer outf.Close()
	outz := gzip.NewWriter(outf)
	defer outz.Close()
	outc := csv.NewWriter(outz)
	defer outc.Flush()

	fields := []string{"TagID", "Time", "CSN", "Room1", "Room2", "Person", "Provider", "UMid",
		"Signal1", "Signal2", "Room_HMM", "Match"}
	outc.Write(fields)

	room := make(map[rfid.RoomCode]string)
	for k, v := range rfid.IPcode {
		room[v] = rfid.IPmap[k]
	}

	for {
		fields = fields[0:0]

		var r rfid.Location

		err := dec.Decode(&r)
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

		fields = append(fields, fmt.Sprintf("%d", r.TagId))
		fields = append(fields, fmt.Sprintf("%s", r.TimeStamp.Format("2006-01-02T15:04:05")))
		fields = append(fields, fmt.Sprintf("%d", r.CSN))
		fields = append(fields, room[r.IP])
		fields = append(fields, room[r.IP2])

		if r.PersonCat == rfid.Patient {
			fields = append(fields, "Patient")
		} else {
			fields = append(fields, "Provider")
		}

		fields = append(fields, rfid.ProvMap[r.ProviderCat])
		fields = append(fields, fmt.Sprintf("%d", r.UMid))

		fields = append(fields, fmt.Sprintf("%f", r.Signal))
		fields = append(fields, fmt.Sprintf("%f", r.Signal2))

		fields = append(fields, room[r.IPhmm])

		if r.Match {
			fields = append(fields, "T")
		} else {
			fields = append(fields, "F")
		}

		outc.Write(fields)
	}
}
