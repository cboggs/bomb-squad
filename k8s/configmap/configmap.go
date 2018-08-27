package configmap

import (
	promcfg "github.com/prometheus/prometheus/config"
	"k8s.io/client-go/kubernetes"
	kcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// Begin proper k8s bits
// ConfigMapWrapper is a struct with public fields, which implements github.com/Fresh-Tracks/bomb-squad/config.Configurator
type ConfigMapWrapper struct {
	// ConfigMapInterface is a client, not the ConfigMap itself
	Client kcorev1.ConfigMapInterface
	Name   string
}

// NewConfigMapWrapper returns a ConfigMapWrapper
func NewConfigMapWrapper(client kubernetes.Interface, namespace string, configMapName string) *ConfigMapWrapper {
	return &ConfigMapWrapper{
		Client: client.CoreV1().ConfigMaps(namespace),
		Name:   configMapName,
	}
}

// Read implements github.com/Fresh-Tracks/bomb-squad/config.Configurator
func (c *ConfigMapWrapper) Read() promcfg.Config {
	return promcfg.Config{}
}

// Write implements github.com/Fresh-Tracks/bomb-squad/config.Configurator
func (c *ConfigMapWrapper) Write([]byte) error {
	return nil
}
