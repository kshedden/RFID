patient_locations.gob.gz:
	go run process_rfid.go

provider_locations.gob.gz:
	go run process_rfid.fo

patient_locations_s.gob.gz: patient_locations.gob.gz
	go run smooth_locs.go patient_locations.gob.gz

provider_locations_s.gob.gz: provider_locations.gob.gz
	go run smooth_locs.go provider_locations.gob.gz

patient_locations_sm.gob.gz: patient_locations_s.gob.gz provider_locations_s.gob.gz
	go run match_locs.go

patient_locations_sm.csv.gz: patient_locations_sm.gob.gz
	go run locstocsv.go patient_locations_sm.gob.gz

provider_locations_sm.csv.gz: provider_locations_sm.gob.gz
	go run locstocsv.go provider_locations_sm.gob.gz

all: patient_locations_sm.csv.gz provider_locations_sm.csv.gz