package rfid

import (
	"math"
	"time"
)

const (
	// Aggregate the signals over this time window.
	twindow time.Duration = 1 * time.Minute
)

// Location describes the predicted location for a person at a given minute.
type Location struct {

	// The id of the tag being located
	TagId uint64

	// The time for the location prediction
	TimeStamp time.Time

	// The numeric code for the location with highest signal
	IP RoomCode

	// The highest signal strength
	Signal float64

	// The numeric code for the location with second highest signal
	IP2 RoomCode

	// The second highest signal
	Signal2 float64

	// The Clarity record number for the appointment
	CSN uint64

	// Indicates whether the tag is held by a provider or a patient
	PersonCat PersonType

	// The provider category holding the tag, if this is held by a provider
	ProviderCat ProviderType

	// The UM id of a provider, 0 if this is a person
	UMid uint64

	// The predicted location from an HMM
	IPhmm RoomCode

	// True if the provider is in a room with a patient, or if a patient is
	// in a room with a provider.
	Match bool
}

// GetLocation returns an array of location predictions corresponding to the provided RFID records.
func GetLocation(recs []*RFIDrecord) []*Location {

	var alocs []*Location

	// Process each minute as a chunk.
	for ii := 0; ii < len(recs); {

		t0 := recs[ii].TimeStamp.Truncate(twindow)

		// Step through to the end of the minute
		jj := ii + 1
		for jj < len(recs) && t0.Equal(recs[jj].TimeStamp.Truncate(twindow)) {
			jj++
		}

		locs := processMinute(recs[ii:jj])
		alocs = append(alocs, locs...)

		ii = jj
	}

	return alocs
}

// argmax2 returns the indices of the two largest values in the array.
// If the array has length 0 or 1, -1's are returned.
func argmax2(x *[32]float64) (int, int) {

	if len(x) == 0 {
		return -1, -1
	}
	if len(x) == 1 {
		return 0, -1
	}

	j0, v0 := 0, x[0]
	j1, v1 := 1, x[1]
	if v1 > v0 {
		j0, j1 = j1, j0
		v0, v1 = v1, v0
	}
	for i := 2; i < len(x); i++ {
		y := x[i]
		if y > v0 {
			j0, j1 = i, j0
			v0, v1 = y, v0
		} else if y > v1 {
			j1 = i
			v1 = y
		}
	}

	return j0, j1
}

// processMinute takes all the RFID records for a single minute and assigns a location
// to each tag id for this minute.
func processMinute(recs []*RFIDrecord) []*Location {

	// 32 = max number of locations
	signal := make(map[uint64]*[32]float64)

	// Map from tag id values to an associated RFID record.  This is only used to get
	// some static meta-data about each tag, so only one record is stored for each tag.
	ctx := make(map[uint64]*RFIDrecord)

	// Get the total signal for each tag within each room.
	for _, x := range recs {
		v, ok := signal[x.TagId]
		if !ok {
			v = new([32]float64)
			signal[x.TagId] = v
		}

		// Update signal
		v[x.IP] += math.Exp(float64(x.Signal) / 10)

		// Update ctx
		ctx[x.TagId] = x
	}

	t0 := recs[0].TimeStamp.Truncate(twindow)

	var locs []*Location
	for tagid, v := range signal {

		// The best and scond best match
		j0, j1 := argmax2(v)

		loc := &Location{
			TagId:       tagid,
			TimeStamp:   t0,
			IP:          RoomCode(j0),
			Signal:      v[j0],
			CSN:         ctx[tagid].CSN,
			IP2:         Null,
			PersonCat:   ctx[tagid].PersonCat,
			ProviderCat: ctx[tagid].ProviderCat,
			UMid:        ctx[tagid].UMid,
		}

		// If there is a second-best match, include it too
		if j1 != -1 {
			loc.IP2 = RoomCode(j1)
			loc.Signal2 = v[j1]
		}
		locs = append(locs, loc)
	}

	return locs
}
