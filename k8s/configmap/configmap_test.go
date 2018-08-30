package configmap

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	k8sTesting "k8s.io/client-go/testing"
)

func TestCanWriteConfigMapConfigMap(t *testing.T) {
	cmw := ConfigMapWrapper{
		Client:  &fakeCoreV1.FakeConfigMaps{Fake: &fakeCoreV1.FakeCoreV1{&k8sTesting.Fake{}}},
		Name:    "testConfigMap",
		DataKey: "testDataKey",
	}

	t.Logf("cmw: %+v", cmw)
	cm, err := cmw.Client.Get("testNamespace", v1.GetOptions{})
	require.NoError(t, err)
	require.Empty(t, cm)
}

func fakeConfigMapClient() *fakeCoreV1.FakeConfigMaps {
	testingFake := &k8sTesting.Fake{}
	testingFakeCoreV1 := &fakeCoreV1.FakeCoreV1{testingFake}
	return &fakeCoreV1.FakeConfigMaps{Fake: testingFakeCoreV1}
}
