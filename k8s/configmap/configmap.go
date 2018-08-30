package configmap

import (
	"fmt"
	"log"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/util/retry"
)

// Begin proper k8s bits
// ConfigMapWrapper is a struct with public fields, which implements github.com/Fresh-Tracks/bomb-squad/config.Configurator
type ConfigMapWrapper struct {
	// ConfigMapInterface is a client, not the ConfigMap itself
	Client  kcorev1.ConfigMapInterface
	Name    string
	DataKey string
}

// NewConfigMapWrapper returns a ConfigMapWrapper
func NewConfigMapWrapper(client kubernetes.Interface, namespace string, configMapName string, dataKey string) *ConfigMapWrapper {
	return &ConfigMapWrapper{
		Client:  client.CoreV1().ConfigMaps(namespace),
		Name:    configMapName,
		DataKey: dataKey,
	}
}

// GetLocation implements github.com/Fresh-Tracks/bomb-squad/config.Configurator
func (c *ConfigMapWrapper) GetLocation() string {
	return c.DataKey
}

// Read implements github.com/Fresh-Tracks/bomb-squad/config.Configurator
func (c *ConfigMapWrapper) Read() []byte {
	dataKey := c.GetLocation()
	cm, err := c.Client.Get(c.Name, v1.GetOptions{})
	if err != nil {
		log.Printf("Failed to get ConfigMap in preparation for Configurator.Read(): %s\n", err)
		return []byte{}
	}

	d := cm.Data[dataKey]

	return []byte(d)
}

// Write implements github.com/Fresh-Tracks/bomb-squad/config.Configurator
func (c *ConfigMapWrapper) Write(data []byte) error {

	dataKey := c.GetLocation()
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Retrieve the latest version of ConfigMap before attempting update
		// RetryOnConflict uses exponential backoff to avoid exhausting the apiserver

		cm, err := c.Client.Get(c.Name, v1.GetOptions{})
		if err != nil {
			panic(fmt.Errorf("Failed to get latest version of ConfigMap: %v", err))
		}

		cm.Data[dataKey] = string(data)

		_, updateErr := c.Client.Update(cm)
		if updateErr != nil {
			return fmt.Errorf("ConfigMap update failed: %v", updateErr)
		}

		return updateErr
	})

	if retryErr != nil {
		return fmt.Errorf("ConfigMap update failed: %v", retryErr)
	}
	log.Println("Updated ConfigMap")

	return nil
}
