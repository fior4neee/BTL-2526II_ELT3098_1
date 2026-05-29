package main

import "math"

const (
	boltzmannDbwPerHzK = -228.6
	defaultFrequencyGhz = 12.0
	defaultEirpDbw = 48.0
	receiverGtDb = 11.5
	bandwidthHz = 25_000_000.0
	bitRateBps = 50_000_000.0
	phasedArrayElements = 256
)

func calcLookAngles(user Location, sat Location) (azimuthDeg, elevationDeg, rangeKm float64) {
	lat1 := toRadians(user.Latitude)
	lon1 := toRadians(user.Longitude)
	lat2 := toRadians(sat.Latitude)
	lon2 := toRadians(sat.Longitude)

	dLon := lon2 - lon1
	centralAngle := math.Acos(math.Sin(lat1)*math.Sin(lat2) + math.Cos(lat1)*math.Cos(lat2)*math.Cos(dLon))
	radiusRatio := 6371.0 / (6371.0 + sat.Altitude)
	elevationRad := math.Atan((math.Cos(centralAngle) - radiusRatio) / math.Max(math.Sin(centralAngle), 1e-9))

	y := math.Sin(dLon) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(dLon)
	azimuth := math.Atan2(y, x)

	rangeKm = math.Sqrt(6371.0*6371.0 + math.Pow(6371.0+sat.Altitude, 2) - 2*6371.0*(6371.0+sat.Altitude)*math.Cos(centralAngle))

	return radToDeg(azimuth), radToDeg(elevationRad), rangeKm
}

func freeSpacePathLossDb(rangeKm, frequencyGhz float64) float64 {
	return 92.45 + 20*math.Log10(rangeKm) + 20*math.Log10(frequencyGhz)
}

func phasedArrayGainDb(elementCount int, elevationDeg float64) float64 {
	apertureGain := 10 * math.Log10(float64(elementCount))
	scanLoss := -3 * (1 - math.Sin(toRadians(elevationDeg)))
	if scanLoss < -12 {
		scanLoss = -12
	}
	return apertureGain + scanLoss
}

func qpskBer(ebN0Db float64) float64 {
	linear := math.Pow(10, ebN0Db/10)
	ber := 0.5 * math.Exp(-linear)
	if ber < 1e-9 {
		return 1e-9
	}
	if ber > 0.5 {
		return 0.5
	}
	return ber
}

func hzToDb(value float64) float64 {
	if value <= 1 {
		value = 1
	}
	return 10 * math.Log10(value)
}

func toRadians(deg float64) float64 {
	return deg * math.Pi / 180
}

func calcLinkMetrics(rangeKm, elevationDeg float64) (cnDb, ebN0Db, pathLossDb, carrierPowerDbm float64) {
	pathLossDb = freeSpacePathLossDb(rangeKm, defaultFrequencyGhz)
	antGain := phasedArrayGainDb(phasedArrayElements, elevationDeg)
	carrierPowerDbm = defaultEirpDbw + antGain - pathLossDb + 30
	cnDb = defaultEirpDbw - pathLossDb + receiverGtDb - boltzmannDbwPerHzK - hzToDb(bandwidthHz)
	ebN0Db = cnDb + hzToDb(bandwidthHz) - hzToDb(bitRateBps)
	return cnDb, ebN0Db, pathLossDb, carrierPowerDbm
}

func modulationForCn(cnDb float64) string {
	if cnDb > 18 {
		return "16QAM 3/4"
	}
	if cnDb > 13 {
		return "QPSK 5/6"
	}
	if cnDb > 10 {
		return "QPSK 3/4"
	}
	return "QPSK 1/2"
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
