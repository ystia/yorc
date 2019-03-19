// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kubernetes

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

type k8s struct {
	clientset kubernetes.Interface
}

func newTestSimpleK8s() *k8s {
	client := k8s{}
	client.clientset = fake.NewSimpleClientset()
	return &client
}

func TestGetVersionDefault(t *testing.T) {
	k8s := newTestSimpleK8s()
	v, err := getVersion(k8s.clientset)
	if err != nil {
		t.Fatal("getVersion should not raise an error")
	}
	expected := "v0.0.0-master+$Format:%h$"
	if v != expected {
		t.Fatal("getVersion should return " + expected)
	}
}

func TestGetExternalIPAdress(t *testing.T) {
	k8s := newTestSimpleK8s()
	nodeExtIP := "1.2.3.4"
	node := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testNode",
		},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{Type: corev1.NodeExternalIP,
					Address: nodeExtIP,
				},
			},
		},
	}
	k8s.clientset.CoreV1().Nodes().Create(&node)
	ip, err := getExternalIPAdress(k8s.clientset, "testNode")
	if err != nil {
		t.Fatal("should not raise an error when IP is present", err)
	}
	if ip != nodeExtIP {
		t.Fatal("IP returned by function (" + ip + ") should be " + nodeExtIP)
	}
}
