package rfid

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type PersonType uint8

type RoomCode uint8

// Codes for the two possible person types
const (
	Provider PersonType = iota
	Patient
)

const (
	Exam1 RoomCode = iota
	Exam2
	Exam3
	Exam4
	Exam5
	Exam6
	Exam7
	Exam8
	Exam9
	Exam10
	Exam11
	Exam12
	Field1
	Field2
	Field3
	Field4
	Field5
	IOLMaster
	Autorefactor
	Admin
	Checkout
	IPW9
	IPW2
	Treatment
	NoSignal
	CheckoutReturn // In HMM but no IP for this room
	Null           // Must be last, used to mark absent Room2
)

var (
	IPcode = map[string]RoomCode{
		"10.23.69.140":   Exam1,
		"10.23.69.141":   Exam2,
		"10.23.69.142":   Exam3,
		"10.23.69.143":   Exam4,
		"10.23.69.144":   Exam5,
		"10.23.69.145":   Exam6,
		"10.23.69.146":   Exam7,
		"10.23.69.147":   Exam8,
		"10.23.69.148":   Exam9,
		"10.23.69.149":   Exam10,
		"10.23.69.150":   Exam11,
		"10.23.69.151":   Exam12,
		"10.23.69.152":   Field1,
		"10.23.69.153":   Field2,
		"10.23.69.154":   Field3,
		"10.23.69.155":   Field4,
		"10.23.69.156":   Field5,
		"10.23.69.157":   IOLMaster,
		"10.23.69.158":   Autorefactor,
		"10.23.69.159":   Admin,
		"10.23.69.160":   Checkout,
		"10.23.69.161":   IPW9,
		"10.23.69.162":   IPW2,
		"10.23.69:163":   Treatment,
		"NoSignal":       NoSignal,
		"CheckoutReturn": CheckoutReturn,
	}

	// Map from IP address to room name.
	IPmap = map[string]string{
		"10.23.69.140":   "Exam1",
		"10.23.69.141":   "Exam2",
		"10.23.69.142":   "Exam3",
		"10.23.69.143":   "Exam4",
		"10.23.69.144":   "Exam5",
		"10.23.69.145":   "Exam6",
		"10.23.69.146":   "Exam7",
		"10.23.69.147":   "Exam8",
		"10.23.69.148":   "Exam9",
		"10.23.69.149":   "Exam10",
		"10.23.69.150":   "Exam11",
		"10.23.69.151":   "Exam12",
		"10.23.69.152":   "Field1",
		"10.23.69.153":   "Field2",
		"10.23.69.154":   "Field3",
		"10.23.69.155":   "Field4",
		"10.23.69.156":   "Field5",
		"10.23.69.157":   "IOLMaster",
		"10.23.69.158":   "Autorefactor",
		"10.23.69.159":   "Admin",
		"10.23.69.160":   "Checkout",
		"10.23.69.161":   "IPW9",
		"10.23.69.162":   "IPW2",
		"10.23.69:163":   "Treatment",
		"NoSignal":       "NoSignal",
		"CheckoutReturn": "CheckoutReturn",
	}

	PTmap = map[PersonType]string{
		Provider: "Provider",
		Patient:  "Patient",
	}

	ProvMap = map[ProviderType]string{
		Attending:     "Attending",
		Fellow:        "Fellow",
		Resident:      "Resident",
		Technician:    "Technician",
		Assistant:     "Assistant",
		Educator:      "Educator",
		Administrator: "Administrator",
		Clerk:         "Clerk",
		Imaging:       "Imaging",
		Other:         "Other",
	}
)

type ProviderType int

// Codes for the different provider types
const (
	Attending     ProviderType = 1
	Fellow                     = 2
	Resident                   = 3
	Technician                 = 4
	Assistant                  = 5
	Educator                   = 6
	Administrator              = 7
	Clerk                      = 8
	Imaging                    = 9
	Other                      = 99
)

type RFIDrecord struct {

	// Unique id
	Ping uint64

	// The type of person corresponding to this record, either
	// patient or provider.
	PersonCat PersonType

	// The source IP address for the tag
	IP RoomCode

	// The unique id for the tag
	TagId uint64

	// Contact serial number (unique identifier for appointment)
	CSN uint64

	// The data that the tag was issued
	TagIssue time.Time

	// The type of provider holding the tag, 0 if the tag is held
	// by a patient
	ProviderCat ProviderType

	// The UM id of the provider holding the tag, 0 if the tag is
	// held by a provider
	UMid uint64

	// The time at which the ping was detected
	TimeStamp time.Time

	// The signal strength
	Signal float32

	// The number of reads from the tag
	Reads uint16

	// The Clarity record for this appointment, if available.
	Clarity *ClarityRecord
}

func (rec *RFIDrecord) parsePatient(tag string, rfi *RFIDinfo) bool {

	fld := strings.Split(tag, "F")
	if len(fld) != 5 {
		rfi.InvalidPatientTag++
		return false
	}

	var err error

	rec.PersonCat = Patient

	rec.TagId, err = strconv.ParseUint(fld[0], 10, 64)
	if err != nil {
		rfi.InvalidPatientTagId++
		return false
	}

	if fld[1] != "0" {
		rfi.InconsistentTag++
		return false
	}

	rec.CSN, err = strconv.ParseUint(fld[2], 10, 64)
	if err != nil {
		rfi.InvalidPatientCSN++
		return false
	}

	var ok bool
	rec.TagIssue, ok = parseMil(fld[3])
	if !ok {
		rfi.InvalidPatientDate++
		return false
	}

	return true
}

// Parse a time string with format MMDDYYHHMM
func parseMil(mil string) (time.Time, bool) {

	if len(mil) != 10 {
		return time.Time{}, false
	}

	month, err := strconv.Atoi(mil[0:2])
	if err != nil {
		return time.Time{}, false
	}

	day, err := strconv.Atoi(mil[2:4])
	if err != nil {
		return time.Time{}, false
	}

	year, err := strconv.Atoi(mil[4:6])
	if err != nil {
		return time.Time{}, false
	}
	year += 2000

	hour, err := strconv.Atoi(mil[6:8])
	if err != nil {
		return time.Time{}, false
	}

	min, err := strconv.Atoi(mil[8:10])
	if err != nil {
		return time.Time{}, false
	}

	return time.Date(year, time.Month(month), day, hour, min, 0, 0, time.UTC), true
}

func (rec *RFIDrecord) parseProvider(tag string, rfi *RFIDinfo) bool {

	fld := strings.Split(tag, "F")
	if len(fld) != 5 {
		rfi.InvalidProviderTag++
		return false
	}

	var err error

	rec.PersonCat = Provider

	rec.TagId, err = strconv.ParseUint(fld[0], 10, 64)
	if err != nil {
		rfi.InvalidProviderTagId++
		return false
	}

	pt, err := strconv.Atoi(fld[1])
	if err != nil {
		rfi.InvalidProviderType++
		return false
	}
	rec.ProviderCat = ProviderType(pt)

	rec.UMid, err = strconv.ParseUint(fld[2], 10, 64)
	if err != nil {
		rfi.InvalidUMid++
		return false
	}

	if len(fld[3]) != 4 {
		rfi.InvalidTagIssueDate++
		return false
	}

	month, err := strconv.Atoi(fld[3][0:2])
	if err != nil {
		rfi.InvalidTagIssueDate++
		return false
	}

	year, err := strconv.Atoi(fld[3][2:4])
	if err != nil {
		rfi.InvalidTagIssueDate++
		return false
	}
	year += 2000

	rec.TagIssue = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)

	return true
}

// Parse takes a row of raw data, split into text tokens, and uses it
// to populate an RFID tag struct.
func (rec *RFIDrecord) Parse(f []string, rfi *RFIDinfo) bool {

	var err error

	rec.Ping, err = strconv.ParseUint(f[0], 10, 64)
	if err != nil {
		rfi.InvalidPing++
		return false
	}

	// Get the IP address as a numeric code
	c, ok := IPcode[f[1]]
	if !ok {
		// Not a known IP address
		rfi.InvalidIP++
		return false
	}
	rec.IP = c

	switch len(f[2]) {
	case 24:
		// Provider record
		if !rec.parseProvider(f[2], rfi) {
			return false
		}
	case 32:
		// Patient record
		if !rec.parsePatient(f[2], rfi) {
			return false
		}
	default:
		rfi.InvalidTagLength++
		return false
	}

	tm := []byte(f[3])
	tm[10] = 'T'
	tm = append(tm, 'Z')
	rec.TimeStamp, err = time.Parse(time.RFC3339, string(tm))
	if err != nil {
		fmt.Printf("%v\n", err)
		rfi.InvalidTimeStamp++
		return false
	}

	s, err := strconv.ParseFloat(f[4], 64)
	if err != nil {
		rfi.InvalidSignal++
		return false
	}
	rec.Signal = float32(s)

	r, err := strconv.Atoi(f[5])
	if err != nil {
		rfi.InvalidReadCount++
		return false
	}
	rec.Reads = uint16(r)

	return true
}

// RFIDinfo contains summary information obtained after processing the
// data for one complete day.
type RFIDinfo struct {
	FileName             string
	InvalidPing          int
	InvalidIP            int
	InvalidTagLength     int
	InvalidTimeStamp     int
	InvalidUMid          int
	InvalidReadCount     int
	InvalidSignal        int
	InvalidProviderTag   int
	InvalidPatientTag    int
	InvalidPatientTagId  int
	InvalidProviderTagId int
	InvalidPatientCSN    int
	InvalidProviderCSN   int
	InvalidProviderType  int
	InvalidTagIssueDate  int
	InvalidPatientDate   int
	InconsistentTag      int
	TotalRecs            int
	FinalRecs            int
	TimeEarly            int
	TimeLate             int
	NoClarity            int
	BeforeCheckin        int
	TimeSpanFull         int
}
