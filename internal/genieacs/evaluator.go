package genieacs

// EvaluateOpticalStatus determines whether an optical Rx reading is normal, warning, critical, or unknown
func EvaluateOpticalStatus(rx *float64, thresholds OpticalThresholds) OpticalStatus {
	if rx == nil {
		return OpticalStatusUnknown
	}

	val := *rx

	// Sinyal overload / terlalu kuat (> -8 dBm) berbahaya untuk receiver optik
	if val > thresholds.NormalMax {
		return OpticalStatusCritical
	}

	// Normal: antara -25 dBm s/d -8 dBm
	if val >= thresholds.NormalMin && val <= thresholds.NormalMax {
		return OpticalStatusNormal
	}

	// Warning: antara -27 dBm s/d -25 dBm
	if val >= thresholds.WarningMin && val < thresholds.NormalMin {
		return OpticalStatusWarning
	}

	// Kritis (redaman terlalu tinggi): di bawah -27 dBm
	if val < thresholds.CriticalMin {
		return OpticalStatusCritical
	}

	return OpticalStatusUnknown
}

