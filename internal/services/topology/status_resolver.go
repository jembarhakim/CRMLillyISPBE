package topology

import "fmt"

// Chain adalah jalur fisik dari pelanggan naik ke server.
//
// Urutan slice dari bawah ke atas: drop, distribusi, backbone. Boleh kosong
// bila topologi belum dipetakan (resolver hanya memakai status ONU).
type Chain struct {
	// Drop: kabel ODP -> pelanggan.
	Drop      *LinkState
	DropLabel string
	// Distribution: POP -> ODP.
	Distribution      *LinkState
	DistributionLabel string
	// Backbone: server -> POP.
	Backbone      *LinkState
	BackboneLabel string
	// ServerName untuk hop terakhir (default "Server NOC").
	ServerName string
}

// Resolve menentukan status efektif satu pelanggan.
//
// Urutan evaluasi WAJIB sama dengan referensi:
//  1. Link ancestor putus  -> down (penyebab infrastruktur lebih diutamakan)
//  2. ONU tidak up (LOS)   -> down
//  3. Rx nil               -> up, "redaman belum terdata" (BUKAN bad)
//  4. Rx < BadMin          -> bad
//  5. Rx < WarnMin         -> warn
//  6. selain itu           -> up
func Resolve(onu OnuState, chain Chain, th Thresholds) Result {
	if chain.ServerName == "" {
		chain.ServerName = "Server NOC"
	}

	// 1. Infrastruktur di atas pelanggan: backbone dulu (dampak terluas),
	//    lalu distribusi, lalu drop. Urutan ini membuat reason menyebut
	//    penyebab paling hulu, sama seperti referensi.
	if chain.Backbone != nil && !chain.Backbone.Up {
		return Result{
			Category: CategoryDown,
			Reason:   backboneLabelOr(chain.BackboneLabel, "Link backbone putus"),
			Source:   sourceOf(chain.Backbone.Source),
			Hops:     buildHops(onu, chain),
		}
	}
	if chain.Distribution != nil && !chain.Distribution.Up {
		return Result{
			Category: CategoryDown,
			Reason:   distributionOrLabel(chain.DistributionLabel, "Link distribusi putus"),
			Source:   sourceOf(chain.Distribution.Source),
			Hops:     buildHops(onu, chain),
		}
	}
	if chain.Drop != nil && !chain.Drop.Up {
		return Result{
			Category: CategoryDown,
			Reason:   "Kabel drop ke pelanggan putus",
			Source:   sourceOf(chain.Drop.Source),
			Hops:     buildHops(onu, chain),
		}
	}

	// 2. ONU sendiri.
	if !onu.OwnUp {
		return Result{
			Category: CategoryDown,
			Reason:   "ONU LOS / mati listrik",
			Source:   sourceOf(onu.Source),
			Hops:     buildHops(onu, chain),
		}
	}

	// 3. Redaman belum terdata — jangan dianggap buruk.
	if onu.Rx == nil {
		return Result{
			Category: CategoryUp,
			Reason:   "Online (redaman belum terdata)",
			Source:   sourceOf(onu.Source),
			Hops:     buildHops(onu, chain),
		}
	}

	rx := *onu.Rx
	res := Result{
		Rx:     onu.Rx,
		Source: sourceOf(onu.Source),
		Hops:   buildHops(onu, chain),
	}

	// 4-6. Klasifikasi redaman.
	switch {
	case th.OverloadMax != 0 && rx > th.OverloadMax:
		res.Category = CategoryBad
		res.Reason = "Sinyal optik overload (terlalu kuat)"
	case rx < th.BadMin:
		res.Category = CategoryBad
		res.Reason = "Redaman melebihi batas"
	case rx < th.WarnMin:
		res.Category = CategoryWarn
		res.Reason = "Redaman mendekati batas"
	default:
		res.Category = CategoryUp
		res.Reason = "Normal"
	}

	return res
}

// backboneLabelOr mengembalikan label khusus backbone bila ada.
func backboneLabelOr(label, fallback string) string {
	if label != "" {
		return "Link backbone ke " + label + " putus"
	}
	return fallback
}

// distributionOrLabel mengembalikan label khusus distribusi bila ada.
func distributionOrLabel(label, fallback string) string {
	if label != "" {
		return "Link distribusi ke " + label + " putus"
	}
	return fallback
}

// sourceOf menormalkan nilai kosong menjadi SourceUnknown.
func sourceOf(s Source) Source {
	if s == "" {
		return SourceUnknown
	}
	return s
}

// buildHops menyusun jalur ONU -> drop -> distribusi -> backbone -> server.
func buildHops(onu OnuState, chain Chain) []Hop {
	hops := make([]Hop, 0, 5)

	hops = append(hops, Hop{
		Kind:   "onu",
		Label:  "ONU pelanggan",
		OK:     onu.OwnUp,
		Source: sourceOf(onu.Source),
	})

	if chain.Drop != nil {
		hops = append(hops, Hop{
			Kind:     "drop",
			Label:    labelOrDefault(chain.DropLabel, "Kabel drop"),
			OK:       chain.Drop.Up,
			Source:   sourceOf(chain.Drop.Source),
			LinkCode: "",
		})
	}
	if chain.Distribution != nil {
		hops = append(hops, Hop{
			Kind:   "distribution",
			Label:  labelOrDefault(chain.DistributionLabel, "Link distribusi"),
			OK:     chain.Distribution.Up,
			Source: sourceOf(chain.Distribution.Source),
		})
	}
	if chain.Backbone != nil {
		hops = append(hops, Hop{
			Kind:   "backbone",
			Label:  labelOrDefault(chain.BackboneLabel, "Link backbone"),
			OK:     chain.Backbone.Up,
			Source: sourceOf(chain.Backbone.Source),
		})
	}

	hops = append(hops, Hop{
		Kind:   "server",
		Label:  chain.ServerName,
		OK:     true,
		Source: SourceUnknown,
	})

	return hops
}

func labelOrDefault(label, fallback string) string {
	if label != "" {
		return label
	}
	return fallback
}

// LabelForCategory mengembalikan label manusiawi untuk kategori.
func LabelForCategory(c Category) string {
	switch c {
	case CategoryUp:
		return "Online"
	case CategoryWarn:
		return "Redaman tinggi"
	case CategoryBad:
		return "Redaman buruk"
	case CategoryDown:
		return "Down"
	default:
		return string(c)
	}
}

// Validate memeriksa kewajaran ambang. Dipakai di test & saat load config.
func (t Thresholds) Validate() error {
	if t.WarnMin >= 0 {
		return fmt.Errorf("WarnMin harus negatif (dBm), didapat %.2f", t.WarnMin)
	}
	if t.BadMin >= 0 {
		return fmt.Errorf("BadMin harus negatif (dBm), didapat %.2f", t.BadMin)
	}
	if t.BadMin > t.WarnMin {
		return fmt.Errorf("BadMin (%.2f) harus lebih kecil dari WarnMin (%.2f)", t.BadMin, t.WarnMin)
	}
	return nil
}
