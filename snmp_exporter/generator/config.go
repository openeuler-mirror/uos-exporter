package main

import (
	"fmt"
	"strconv"

	"snmp_exporter/internal/metrics"
)

// The generator metrics.
type Config struct {
	Auths   map[string]*metrics.Auth `yaml:"auths"`
	Modules map[string]*ModuleConfig `yaml:"modules"`
	Version int                      `yaml:"version,omitempty"`
}

type MetricOverrides struct {
	Ignore          bool                               `yaml:"ignore,omitempty"`
	RegexpExtracts  map[string][]metrics.RegexpExtract `yaml:"regex_extracts,omitempty"`
	DateTimePattern string                             `yaml:"datetime_pattern,omitempty"`
	Offset          float64                            `yaml:"offset,omitempty"`
	Scale           float64                            `yaml:"scale,omitempty"`
	Type            string                             `yaml:"type,omitempty"`
	Help            string                             `yaml:"help,omitempty"`
	Name            string                             `yaml:"name,omitempty"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *MetricOverrides) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain MetricOverrides
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	// Ensure type for override is valid if one is defined.
	typ, ok := metricType(c.Type)
	if c.Type != "" && (!ok || typ != c.Type) {
		return fmt.Errorf("invalid metric type override '%s'", c.Type)
	}

	return nil
}

type ModuleConfig struct {
	Walk       []string                   `yaml:"walk"`
	Lookups    []*Lookup                  `yaml:"lookups"`
	WalkParams metrics.WalkParams         `yaml:",inline"`
	Overrides  map[string]MetricOverrides `yaml:"overrides"`
	Filters    metrics.Filters            `yaml:"filters,omitempty"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *ModuleConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain ModuleConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}

	// Ensure indices in static filters are integer for input validation.
	for _, filter := range c.Filters.Static {
		for _, index := range filter.Indices {
			_, err := strconv.Atoi(index)
			if err != nil {
				return fmt.Errorf("invalid index '%s'. Index must be integer", index)
			}
		}
	}

	return nil
}

type Lookup struct {

// TODO: implement functions
