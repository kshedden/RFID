package rfid

import "time"

// ClarityRecord contains extracted fields for one record of Clarity data.
type ClarityRecord struct {

	// Appointment identifier
	CSN uint64

	// Time of check-in
	CheckInTime time.Time

	// Time of check-out
	CheckOutTime time.Time

	// Provider name
	ProvName string

	// Visual field
	VfiOs float64
}

// ClarityFileInfo contains the column positions for the variables of interest.
type ClarityFileInfo struct {

	// Column positions
	CSN          int
	CheckInTime  int
	CheckOutTime int
	ProvName     int
	VfiOs        int
}

// GetClarityFileInfo takes the header of a Clarity file and locates the columns of interest.
func GetClarityFileInfo(head []string) *ClarityFileInfo {

	col := make(map[string]int)
	for j, n := range head {
		col[n] = j
	}

	finf := new(ClarityFileInfo)

	var ok bool

	finf.CSN, ok = col["PAT_ENC_CSN_ID"]
	if !ok {
		panic("Can't find PAT_ENC_CSN_ID\n")
	}

	finf.CheckInTime, ok = col["CHECKIN_DTTM"]
	if !ok {
		panic("Can't find CHECKIN_DTTM\n")
	}

	finf.CheckOutTime, ok = col["CHECKOUT_DTTM"]
	if !ok {
		panic("Can't find CHECKOUT_DTTM\n")
	}

	finf.ProvName, ok = col["PROV_NAME_WID"]
	if !ok {
		panic("Can't find PROV_NAME_WID\n")
	}

	finf.VfiOs, ok = col["VFI_OS"]
	if !ok {
		panic("Cant find VFI_OS")
	}

	return finf
}
