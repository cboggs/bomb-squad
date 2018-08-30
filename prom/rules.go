package prom

import (
	"log"

	"github.com/Fresh-Tracks/bomb-squad/config"
	promcfg "github.com/prometheus/prometheus/config"
	yaml "gopkg.in/yaml.v2"
)

// AppendRuleFile Appends a static rule file that Bomb Squad needs into the
// array of rule files that may exist in the current Prometheus config
func AppendRuleFile(filename string, c config.Configurator) error {
	cfg := config.ReadPromConfig(c)
	configRuleFiles := cfg.RuleFiles
	ruleFileFound := false

	for _, f := range configRuleFiles {
		if f == filename {
			ruleFileFound = true
		}
	}

	if !ruleFileFound {
		newRuleFiles := append(configRuleFiles, filename)
		cfg.RuleFiles = newRuleFiles
		err := config.WritePromConfig(cfg, c)
		if err != nil {
			return err
		}
	}
	return nil
}

// ReUnmarshal simply marshals a RelabelConfig and unmarshals it again back into place.
// This is needed to accomodate an "expansion", if you will, of the prometheus.config
// Regexp struct's string representation that happens only upon unmarshalling it.
// TODO: (TODON'T?) Instead of this, figure out the unmarshalling quirk and change it
func ReUnmarshal(rc *promcfg.RelabelConfig) {
	s, err := yaml.Marshal(rc)
	if err != nil {
		log.Fatal(err)
	}
	err = yaml.Unmarshal(s, rc)
	if err != nil {
		log.Fatal(err)
	}
}
