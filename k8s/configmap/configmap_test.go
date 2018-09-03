package configmap

import (
	"log"
	"testing"

	"github.com/stretchr/testify/require"
	k8sAPICoreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	kCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

func TestCanReadConfigMap(t *testing.T) {
	cmw := NewConfigMapWrapper(fakeConfigMapClient(), "testNamespace", "testConfigMap", "testDataKey")

	cm, err := cmw.Client.Create(newConfigMap())
	if err != nil {
		log.Fatal(err)
	}

	b := cmw.Read()

	require.Equal(t, cm.Data["testDataKey"], string(b))
}

func TestCanWriteConfigMap(t *testing.T) {
	cmw := NewConfigMapWrapper(fakeConfigMapClient(), "testNamespace", "testConfigMap", "testDataKey")

	_, err := cmw.Client.Create(newConfigMap())
	if err != nil {
		log.Fatal(err)
	}

	err = cmw.Write([]byte("BazBat"))
	require.NoError(t, err)

	b := cmw.Read()
	require.Equal(t, "BazBat", string(b))
}

func fakeConfigMapClient() kCoreV1.ConfigMapInterface {

	return fake.NewSimpleClientset().CoreV1().ConfigMaps("testNamespace")
}

func newConfigMap() *k8sAPICoreV1.ConfigMap {
	cmType := metaV1.TypeMeta{
		Kind:       "ConfigMap",
		APIVersion: "core/v1",
	}
	cmMeta := metaV1.ObjectMeta{
		Name:      "testConfigMap",
		Namespace: "testNamespace",
	}
	cmData := map[string]string{
		"testDataKey": "FooBar",
	}

	return &k8sAPICoreV1.ConfigMap{TypeMeta: cmType, ObjectMeta: cmMeta, Data: cmData}
}
