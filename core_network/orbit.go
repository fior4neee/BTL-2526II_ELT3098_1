package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	earthMuKm3S2 = 398600.4418
)

// TleRecord represents a single satellite TLE entry
// This is a simplified propagator for demo use (Keplerian elements).
type TleRecord struct {
	Name         string
	Line1        string
	Line2        string
	Epoch        time.Time
	Inclination  float64
	RAAN         float64
	Eccentricity float64
	ArgPerigee   float64
	MeanAnomaly  float64
	MeanMotion   float64
}

// SatelliteTracker maintains current satellite states from TLE data
// Updated at a fixed interval by TelemetryEngine.
type SatelliteTracker struct {
	mu      sync.RWMutex
	records []TleRecord
	states []Satellite
}

func NewSatelliteTracker(records []TleRecord) *SatelliteTracker {
	return &SatelliteTracker{records: records}
}

func (st *SatelliteTracker) Update(now time.Time) {
	states := make([]Satellite, 0, len(st.records))
	for _, rec := range st.records {
		loc, err := propagateTle(rec, now)
		if err != nil {
			continue
		}
		states = append(states, Satellite{
			ID:        rec.Name,
			Name:      rec.Name,
			Location:  loc,
			Latitude:  loc.Latitude,
			Longitude: loc.Longitude,
			Altitude:  loc.Altitude,
			Elevation: 0,
			Azimuth:   0,
			Timestamp: now,
		})
	}

	st.mu.Lock()
	st.states = states
	st.mu.Unlock()
}

func (st *SatelliteTracker) GetStates() []Satellite {
	st.mu.RLock()
	defer st.mu.RUnlock()
	res := make([]Satellite, len(st.states))
	copy(res, st.states)
	return res
}

func LoadTLE(path string) ([]TleRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var records []TleRecord
	scanner := bufio.NewScanner(file)
	var name, line1 string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "1 ") {
			line1 = line
			continue
		}
		if strings.HasPrefix(line, "2 ") && line1 != "" {
			rec, err := parseTleRecord(name, line1, line)
			if err == nil {
				records = append(records, rec)
			}
			name = ""
			line1 = ""
			continue
		}
		// Name line
		name = line
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("no TLE records found")
	}
	return records, nil
}

func parseTleRecord(name, line1, line2 string) (TleRecord, error) {
	if len(line1) < 32 || len(line2) < 63 {
		return TleRecord{}, fmt.Errorf("invalid TLE length")
	}
	epochRaw := strings.TrimSpace(line1[18:32])
	if len(epochRaw) < 5 {
		return TleRecord{}, fmt.Errorf("invalid epoch")
	}
	yearPart, _ := strconv.Atoi(epochRaw[:2])
	dayPart, _ := strconv.ParseFloat(epochRaw[2:], 64)
	year := 2000 + yearPart
	if yearPart >= 57 {
		year = 1900 + yearPart
	}
	epoch := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).
		Add(time.Duration((dayPart-1)*24*float64(time.Hour)))

	inclination := parseFloat(line2[8:16])
	raan := parseFloat(line2[17:25])
	ecc := parseFloat("0." + strings.TrimSpace(line2[26:33]))
	argPerigee := parseFloat(line2[34:42])
	meanAnomaly := parseFloat(line2[43:51])
	meanMotion := parseFloat(line2[52:63])

	return TleRecord{
		Name:         strings.TrimSpace(name),
		Line1:        line1,
		Line2:        line2,
		Epoch:        epoch,
		Inclination:  inclination,
		RAAN:         raan,
		Eccentricity: ecc,
		ArgPerigee:   argPerigee,
		MeanAnomaly:  meanAnomaly,
		MeanMotion:   meanMotion,
	}, nil
}

func parseFloat(raw string) float64 {
	val, _ := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	return val
}

func propagateTle(rec TleRecord, now time.Time) (Location, error) {
	n := rec.MeanMotion * 2 * math.Pi / 86400.0
	if n <= 0 {
		return Location{}, fmt.Errorf("invalid mean motion")
	}

	a := math.Pow(earthMuKm3S2/(n*n), 1.0/3.0)
	dt := now.Sub(rec.Epoch).Seconds()
	M := degToRad(rec.MeanAnomaly) + n*dt
	M = math.Mod(M, 2*math.Pi)

	E := solveKepler(M, rec.Eccentricity)
	v := 2 * math.Atan2(math.Sqrt(1+rec.Eccentricity)*math.Sin(E/2), math.Sqrt(1-rec.Eccentricity)*math.Cos(E/2))
	r := a * (1 - rec.Eccentricity*math.Cos(E))

	xp := r * math.Cos(v)
	yp := r * math.Sin(v)

	cosO := math.Cos(degToRad(rec.RAAN))
	sinO := math.Sin(degToRad(rec.RAAN))
	cosI := math.Cos(degToRad(rec.Inclination))
	sinI := math.Sin(degToRad(rec.Inclination))
	cosw := math.Cos(degToRad(rec.ArgPerigee))
	sinw := math.Sin(degToRad(rec.ArgPerigee))

	// Perifocal to ECI
	x := (cosO*cosw - sinO*sinw*cosI)*xp + (-cosO*sinw - sinO*cosw*cosI)*yp
	y := (sinO*cosw + cosO*sinw*cosI)*xp + (-sinO*sinw + cosO*cosw*cosI)*yp
	z := (sinw*sinI)*xp + (cosw*sinI)*yp

	gmst := greenwichMeanSiderealTime(now)
	cosG := math.Cos(gmst)
	sinG := math.Sin(gmst)

	// ECI to ECEF rotation
	xE := cosG*x + sinG*y
	yE := -sinG*x + cosG*y
	zE := z

	lon := math.Atan2(yE, xE)
	lat := math.Atan2(zE, math.Sqrt(xE*xE+yE*yE))
	radius := math.Sqrt(xE*xE + yE*yE + zE*zE)
	alt := radius - 6371.0

	return Location{
		Latitude:  radToDeg(lat),
		Longitude: radToDeg(lon),
		Altitude:  alt,
	}, nil
}

func solveKepler(M, e float64) float64 {
	E := M
	for i := 0; i < 8; i++ {
		f := E - e*math.Sin(E) - M
		fPrime := 1 - e*math.Cos(E)
		E -= f / fPrime
	}
	return E
}

func degToRad(deg float64) float64 {
	return deg * math.Pi / 180
}

func radToDeg(rad float64) float64 {
	return rad * 180 / math.Pi
}

func greenwichMeanSiderealTime(t time.Time) float64 {
	jd := julianDate(t.UTC())
	T := (jd - 2451545.0) / 36525.0
	gmst := 280.46061837 + 360.98564736629*(jd-2451545.0) + 0.000387933*T*T - (T*T*T)/38710000.0
	gmst = math.Mod(gmst, 360.0)
	if gmst < 0 {
		gmst += 360.0
	}
	return degToRad(gmst)
}

func julianDate(t time.Time) float64 {
	y := t.Year()
	m := int(t.Month())
	d := float64(t.Day()) + (float64(t.Hour())+float64(t.Minute())/60+float64(t.Second())/3600)/24
	if m <= 2 {
		y -= 1
		m += 12
	}
	A := int(float64(y) / 100.0)
	B := 2 - A + int(float64(A)/4.0)
	jd := math.Floor(365.25*float64(y+4716)) + math.Floor(30.6001*float64(m+1)) + d + float64(B) - 1524.5
	return jd
}
