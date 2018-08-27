package prom

import (
	"context"

	"github.com/Fresh-Tracks/bomb-squad/config"
	yaml "gopkg.in/yaml.v2"
)

// AppendRuleFile Appends a static rule file that Bomb Squad needs into the
// array of rule files that may exist in the current Prometheus config
func AppendRuleFile(ctx context.Context, filename string, pc config.PromConfigurator) error {
	cfg := pc.Read()
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
		cfgBytes, err := yaml.Marshal(cfg)
		err = pc.Write(cfgBytes)
		if err != nil {
			return err
		}
	}
	return nil
}
