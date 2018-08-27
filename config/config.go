package config

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Fresh-Tracks/bomb-squad/util"
	"github.com/prometheus/common/model"
	promcfg "github.com/prometheus/prometheus/config"
	yaml "gopkg.in/yaml.v2"
)

type PromConfigurator interface {
	Read() promcfg.Config
	Write([]byte) error
}

type BSConfigurator interface {
	Read() BombSquadConfig
	Write([]byte) error
}

type BombSquadLabelConfig map[string]string

type BombSquadConfig struct {
	SuppressedMetrics map[string]BombSquadLabelConfig
}

func GetBombSquadConfig(bc BSConfigurator) BombSquadConfig {
	b := bc.Read()
	//	if len(bsConfig) > 0 {
	//		err := yaml.Unmarshal(bsConfig, &b)
	//		if err != nil {
	//			log.Fatal(err)
	//		}
	//	}

	if b.SuppressedMetrics == nil {
		b.SuppressedMetrics = map[string]BombSquadLabelConfig{}
	}

	return b
}

func ListSuppressedMetrics(bc BSConfigurator) {
	b := GetBombSquadConfig(bc)
	for metric, labels := range b.SuppressedMetrics {
		for label := range labels {
			fmt.Printf("%s.%s\n", metric, label)
		}
	}
}

func RemoveSilence(label string, pc PromConfigurator, bc BSConfigurator) error {
	promConfig := pc.Read()

	ml := strings.Split(label, ".")
	metricName, labelName := ml[0], ml[1]

	bsCfg := GetBombSquadConfig(bc)
	bsRelabelConfigEncoded := bsCfg.SuppressedMetrics[metricName][labelName]

	for _, scrapeConfig := range promConfig.ScrapeConfigs {
		i := FindRelabelConfigInScrapeConfig(bsRelabelConfigEncoded, *scrapeConfig)
		if i >= 0 {
			scrapeConfig.MetricRelabelConfigs = DeleteRelabelConfigFromArray(scrapeConfig.MetricRelabelConfigs, i)
			fmt.Printf("Deleted silence rule from ScrapeConfig %s\n", scrapeConfig.JobName)
		}
	}

	if len(bsCfg.SuppressedMetrics[metricName]) == 1 {
		delete(bsCfg.SuppressedMetrics, metricName)
	} else {
		delete(bsCfg.SuppressedMetrics[metricName], labelName)
	}

	bsCfgBytes, err := yaml.Marshal(bsCfg)
	if err != nil {
		log.Fatal(err)
	}

	promConfigBytes, err := yaml.Marshal(promConfig)
	if err != nil {
		log.Fatal(err)
	}

	bc.Write(bsCfgBytes)
	pc.Write(promConfigBytes)

	resetMetric(metricName, labelName)

	return nil
}

func StoreMetricRelabelConfigBombSquad(s HighCardSeries, mrc promcfg.RelabelConfig, bc BSConfigurator) {
	b := GetBombSquadConfig(bc)
	if lc, ok := b.SuppressedMetrics[s.MetricName]; ok {
		lc[string(s.HighCardLabelName)] = Encode(mrc)
	} else {
		b.SuppressedMetrics[s.MetricName] = lc
		lc = BombSquadLabelConfig{}
		lc[string(s.HighCardLabelName)] = Encode(mrc)
	}

	res, err := yaml.Marshal(b)
	if err != nil {
		log.Fatal(err)
	}

	err = bc.Write(res)
	if err != nil {
		log.Fatal(err)
	}
}

func DeleteRelabelConfigFromArray(arr []*promcfg.RelabelConfig, index int) []*promcfg.RelabelConfig {
	res := []*promcfg.RelabelConfig{}
	if len(arr) > 1 {
		res = append(arr[:index], arr[index+1:]...)
	} else {
		res = []*promcfg.RelabelConfig{}
	}
	return res
}

func FindRelabelConfigInScrapeConfig(encodedRule string, scrapeConfig promcfg.ScrapeConfig) int {
	for i, relabelConfig := range scrapeConfig.MetricRelabelConfigs {
		if Encode(*relabelConfig) == encodedRule {
			return i
		}
	}

	return -1
}

func InsertMetricRelabelConfigToPromConfig(rc promcfg.RelabelConfig, pc PromConfigurator) promcfg.Config {
	promConfig := pc.Read()
	rcEncoded := Encode(rc)
	for _, scrapeConfig := range promConfig.ScrapeConfigs {
		if FindRelabelConfigInScrapeConfig(rcEncoded, *scrapeConfig) == -1 {
			fmt.Printf("Did not find necessary silence rule in ScrapeConfig %s, adding now\n", scrapeConfig.JobName)
			scrapeConfig.MetricRelabelConfigs = append(scrapeConfig.MetricRelabelConfigs, &rc)
		}
	}
	return promConfig
}
func Encode(rc promcfg.RelabelConfig) string {
	b, err := yaml.Marshal(rc)
	if err != nil {
		log.Fatal(err)
	}

	s := fmt.Sprintf("%s", string(b))
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func ConfigGetRuleFiles() []string {
	return []string{"nope", "not yet"}
}

// HighCardSeries represents a Prometheus series that has been idenitified as
// high cardinality
type HighCardSeries struct {
	MetricName        string
	HighCardLabelName model.LabelName
}

// TODO: Only generate the relabel config for the appropriate job that is spitting out
// the high-cardinality metric
// TODO: Within a job, some series may never be exploding on this label. Consider including
// all relevant labels in source_labels...?
func GenerateMetricRelabelConfig(s HighCardSeries) promcfg.RelabelConfig {
	valueReplace := "bs_silence"
	regexpOriginal := fmt.Sprintf("^%s;.*$", s.MetricName)
	promRegex, err := promcfg.NewRegexp(regexpOriginal)
	if err != nil {
		log.Fatal(err)
	}

	newMetricRelabelConfig := promcfg.RelabelConfig{
		SourceLabels: model.LabelNames{"__name__", s.HighCardLabelName},
		Regex:        promRegex,
		TargetLabel:  string(s.HighCardLabelName),
		Replacement:  valueReplace,
		Action:       "replace",
	}
	return newMetricRelabelConfig
}

func resetMetric(metricName, labelName string) {
	client, _ := util.HttpClient()
	// What is this localhost 8080?
	endpt := fmt.Sprintf("http://localhost:8080/metrics/reset?metric=%s&label=%s", metricName, labelName)
	req, _ := http.NewRequest("GET", endpt, nil)

	_, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to reset metric for %s.%s: %s. Not urgent - continuing.", metricName, labelName, err)
	}
}
