package prom

import (
	"github.com/Fresh-Tracks/bomb-squad/config"
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
