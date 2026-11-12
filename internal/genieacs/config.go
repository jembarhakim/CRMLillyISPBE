package genieacs

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// OpticalThresholds defines the optical power boundary criteria
type OpticalThresholds struct {
	NormalMin   float64 `yaml:"normal_min"`   // default -25.0 dBm
	NormalMax   float64 `yaml:"normal_max"`   // default -8.0 dBm
	WarningMin  float64 `yaml:"warning_min"`  // default -27.0 dBm
	CriticalMin float64 `yaml:"critical_min"` // default -27.0 dBm
}

// ConfigThresholds holds threshold configuration
type ConfigThresholds struct {
	OpticalRx                OpticalThresholds `yaml:"optical_rx"`
	OfflineTimeoutMultiplier float64           `yaml:"offline_timeout_multiplier"` // default 2.0
	DefaultInformInterval    int               `yaml:"default_inform_interval"`    // default 300 seconds
}

// MetricParam defines parameter path and conversion scale
type MetricParam struct {
	Path         string  `yaml:"path"`
	FallbackPath string  `yaml:"fallback_path"`
	Scale        float64 `yaml:"scale"`
	Unit         string  `yaml:"unit"`
}

// UnmarshalYAML accepts both shapes used inside config/ont-models.yaml:
//
// 1. Object form (vendor profiles):
//    rx_power:
//      path: "InternetGatewayDevice...RXPower"
//      scale: 1.0
//
// 2. Scalar form (fallback candidates), where the value is just the path:
//    ssid_candidates:
//      - "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"
//
// Without this, the parameters map (declared as MetricParam) would reject the
// scalar entries and the whole config would fail to load.
func (m *MetricParam) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		m.Path = value.Value
		if m.Scale == 0 {
			m.Scale = 1.0
		}
		return nil
	}

	// Object form: decode into a shadow struct to avoid recursion
	type plain MetricParam
	var p plain
	if err := value.Decode(&p); err != nil {
		return err
	}
	*m = MetricParam(p)
	if m.Scale == 0 {
		m.Scale = 1.0
	}
	return nil
}

// ModelMatcher defines regex patterns for matching vendor models
type ModelMatcher struct {
	Manufacturer string `yaml:"manufacturer"`
	ProductClass string `yaml:"product_class"`
}

// GenericCandidates defines fallback candidate paths
type GenericCandidates struct {
	RxPowerCandidates     []MetricParam `yaml:"rx_power_candidates"`
	TxPowerCandidates     []MetricParam `yaml:"tx_power_candidates"`
	SSIDCandidates        []string      `yaml:"ssid_candidates"`
	ClientCountCandidates []string      `yaml:"client_count_candidates"`
}

// ModelProfile defines the profile for a specific ONT vendor or model
type ModelProfile struct {
	Key        string                `yaml:"-"`
	Name       string                `yaml:"name"`
	Vendor     string                `yaml:"vendor"`
	Match      ModelMatcher          `yaml:"match"`
	DataModel  string                `yaml:"data_model"`
	Verified   bool                  `yaml:"verified"`
	Parameters map[string]MetricParam `yaml:"parameters"`

	// Candidates holds list-style entries such as rx_power_candidates
	// (used by the generic_fallback profile).
	Candidates map[string][]MetricParam `yaml:"-"`
}

// parameterKeyAliases maps the list-style YAML keys of generic_fallback onto
// the canonical metric names used by the parser.

// UnmarshalYAML decodes a profile, accepting both parameter styles:
//
//   - object style: rx_power: { path: ..., scale: ... }  -> Parameters
//   - list style:   rx_power_candidates: [ "...", ... ]  -> Candidates
func (p *ModelProfile) UnmarshalYAML(value *yaml.Node) error {
	// Decode the scalar fields with an explicit struct (no "parameters" key),
	// so list-style parameter entries never hit the map decoder.
	var shadow struct {
		Name      string       `yaml:"name"`
		Vendor    string       `yaml:"vendor"`
		Match     ModelMatcher `yaml:"match"`
		DataModel string       `yaml:"data_model"`
		Verified  bool         `yaml:"verified"`
	}

	if value.Kind == yaml.MappingNode {
		if err := value.Decode(&shadow); err != nil {
			return err
		}
	}

	p.Name = shadow.Name
	p.Vendor = shadow.Vendor
	p.Match = shadow.Match
	p.DataModel = shadow.DataModel
	p.Verified = shadow.Verified
	p.Parameters = map[string]MetricParam{}
	p.Candidates = map[string][]MetricParam{}

	if value.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i+1 < len(value.Content); i += 2 {
		if value.Content[i].Value != "parameters" {
			continue
		}

		node := value.Content[i+1]
		if node.Kind != yaml.MappingNode {
			continue
		}

		for j := 0; j+1 < len(node.Content); j += 2 {
			key := node.Content[j].Value
			entry := node.Content[j+1]

			if entry.Kind == yaml.SequenceNode {
				list := make([]MetricParam, 0, len(entry.Content))
				if err := entry.Decode(&list); err != nil {
					return fmt.Errorf("failed to decode parameter list %q: %w", key, err)
				}
				p.Candidates[key] = list
				continue
			}

			var param MetricParam
			if err := entry.Decode(&param); err != nil {
				return fmt.Errorf("failed to decode parameter %q: %w", key, err)
			}
			if param.Scale == 0 {
				param.Scale = 1.0
			}
			p.Parameters[key] = param
		}
	}

	return nil
}

// HasMetric reports whether the profile can supply a metric either through the
// object style (rx_power) or the list style (rx_power_candidates).
func (p *ModelProfile) HasMetric(key string) bool {
	if p == nil {
		return false
	}
	if p.HasParameter(key) {
		return true
	}
	return len(p.CandidatePaths(key)) > 0
}

// CandidateParam returns the first candidate entry for a canonical metric name
// (e.g. "rx_power"), including list-style declarations such as rx_power_candidates.
func (p *ModelProfile) CandidateParam(key string) (MetricParam, bool) {
	if p == nil {
		return MetricParam{}, false
	}
	if p.Candidates == nil {
		return MetricParam{}, false
	}

	aliasKey := key + "_candidates"
	if list, ok := p.Candidates[aliasKey]; ok && len(list) > 0 {
		return list[0], true
	}
	if list, ok := p.Candidates[key]; ok && len(list) > 0 {
		return list[0], true
	}
	return MetricParam{}, false
}

// CandidatePaths returns all candidate paths for a canonical metric name.
func (p *ModelProfile) CandidatePaths(key string) []MetricParam {
	if p == nil || p.Candidates == nil {
		return nil
	}
	if list, ok := p.Candidates[key+"_candidates"]; ok {
		return list
	}
	return p.Candidates[key]
}

// HasParameter reports whether the profile declares the given metric (e.g. "rx_power")
func (p *ModelProfile) HasParameter(key string) bool {
	if p == nil || p.Parameters == nil {
		return false
	}
	_, ok := p.Parameters[key]
	return ok
}

// Parameter returns the metric definition for key, or a zero value when absent
func (p *ModelProfile) Parameter(key string) MetricParam {
	if p == nil || p.Parameters == nil {
		return MetricParam{Scale: 1.0}
	}
	if param, ok := p.Parameters[key]; ok {
		if param.Scale == 0 {
			param.Scale = 1.0
		}
		return param
	}
	return MetricParam{Scale: 1.0}
}

// ONTModelsConfig is the root configuration structure
//
// Models is an ordered slice (not a map) so vendor matching is deterministic:
// previously a Go map iteration order was random and the catch-all
// "generic_fallback" profile could win over a specific vendor profile.
// Generic/fallback profiles are always evaluated last.
type ONTModelsConfig struct {
	Thresholds ConfigThresholds `yaml:"thresholds"`
	Models     []ModelProfile   `yaml:"models"`
}

// FindModel returns the profile registered under the given key (e.g. "huawei_tr098")
func (c *ONTModelsConfig) FindModel(key string) *ModelProfile {
	if c == nil {
		return nil
	}
	for i := range c.Models {
		if c.Models[i].Key == key {
			return &c.Models[i]
		}
	}
	return nil
}

// unmarshalModels decodes the YAML "models" mapping while preserving the order
// in which the profiles are declared in the file.
func unmarshalModels(node *yaml.Node) ([]ModelProfile, error) {
	profiles := make([]ModelProfile, 0, len(node.Content)/2)

	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		var profile ModelProfile
		if err := valueNode.Decode(&profile); err != nil {
			return nil, fmt.Errorf("failed to decode model profile %q: %w", keyNode.Value, err)
		}
		profile.Key = keyNode.Value
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

// UnmarshalYAML decodes the root document while keeping the declaration order of models.
func (c *ONTModelsConfig) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("ont-models config must be a mapping at the root")
	}

	for i := 0; i+1 < len(value.Content); i += 2 {
		key := value.Content[i].Value
		node := value.Content[i+1]

		switch key {
		case "thresholds":
			if err := node.Decode(&c.Thresholds); err != nil {
				return fmt.Errorf("failed to decode thresholds: %w", err)
			}
		case "models":
			models, err := unmarshalModels(node)
			if err != nil {
				return err
			}
			c.Models = models
		}
	}

	return nil
}

// DefaultConfig returns safe fallback defaults
func DefaultConfig() *ONTModelsConfig {
	return &ONTModelsConfig{
		Thresholds: ConfigThresholds{
			OpticalRx: OpticalThresholds{
				NormalMin:   -25.0,
				NormalMax:   -8.0,
				WarningMin:  -27.0,
				CriticalMin: -27.0,
			},
			OfflineTimeoutMultiplier: 2.0,
			DefaultInformInterval:    300,
		},
		Models: []ModelProfile{},
	}
}

// LoadConfig loads the ONT model configuration from file with fallback paths
func LoadConfig(customPath string) (*ONTModelsConfig, error) {
	cfg := DefaultConfig()

	candidatePaths := []string{}
	if customPath != "" {
		candidatePaths = append(candidatePaths, customPath)
	}
	if envPath := os.Getenv("ONT_CONFIG_PATH"); envPath != "" {
		candidatePaths = append(candidatePaths, envPath)
	}
	candidatePaths = append(candidatePaths,
		"./config/ont-models.yaml",
		"../config/ont-models.yaml",
		"../../config/ont-models.yaml",
		"CRMLillyISPBE/config/ont-models.yaml",
	)

	var loadedPath string
	for _, p := range candidatePaths {
		clean := filepath.Clean(p)
		if info, err := os.Stat(clean); err == nil && !info.IsDir() {
			loadedPath = clean
			break
		}
	}

	if loadedPath == "" {
		return cfg, fmt.Errorf("config/ont-models.yaml not found in candidate paths, using default thresholds")
	}

	data, err := os.ReadFile(loadedPath)
	if err != nil {
		return cfg, fmt.Errorf("failed to read %s: %w", loadedPath, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return cfg, fmt.Errorf("failed to unmarshal yaml %s: %w", loadedPath, err)
	}

	// Apply safe fallbacks for missing threshold values
	if cfg.Thresholds.OpticalRx.NormalMin == 0 {
		cfg.Thresholds.OpticalRx.NormalMin = -25.0
	}
	if cfg.Thresholds.OpticalRx.NormalMax == 0 {
		cfg.Thresholds.OpticalRx.NormalMax = -8.0
	}
	if cfg.Thresholds.OpticalRx.WarningMin == 0 {
		cfg.Thresholds.OpticalRx.WarningMin = -27.0
	}
	if cfg.Thresholds.OpticalRx.CriticalMin == 0 {
		cfg.Thresholds.OpticalRx.CriticalMin = -27.0
	}
	if cfg.Thresholds.OfflineTimeoutMultiplier <= 0 {
		cfg.Thresholds.OfflineTimeoutMultiplier = 2.0
	}
	if cfg.Thresholds.DefaultInformInterval <= 0 {
		cfg.Thresholds.DefaultInformInterval = 300
	}

	return cfg, nil
}

