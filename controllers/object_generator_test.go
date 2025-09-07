// Copyright 2019 The redis-operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sv1alpha1 "github.com/bringg/redis-operator/api/v1alpha1"
	"github.com/bringg/redis-operator/controllers/redis"
)

func Test_mapsEqual(t *testing.T) {
	var aNil, bNil map[string]string
	type args struct {
		a map[string]string
		b map[string]string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"nil", args{aNil, bNil}, true},
		{"empty", args{map[string]string{}, map[string]string{}}, true},
		{"match", args{map[string]string{"ok": "lol"}, map[string]string{"ok": "lol"}}, true},
		{"no-match", args{map[string]string{"ok": "lol"}, map[string]string{}}, false},
		{"no-match", args{map[string]string{"ok": "lol"}, map[string]string{"wow": "cool"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapsEqual(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("mapsEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isSubset(t *testing.T) {
	type args struct {
		a map[string]string
		b map[string]string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"subset", args{map[string]string{"ok": "lol", "wow": "cool"}, map[string]string{"wow": "cool"}}, true},
		{"no-value-match", args{map[string]string{"ok": "lol", "wow": "cool"}, map[string]string{"wow": "such"}}, false},
		{"no-subset", args{map[string]string{"ok": "lol"}, map[string]string{"wow": "cool"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSubset(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("isSubset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_normalizeResourceRequirements(t *testing.T) {
	input := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("4096Mi"),
			corev1.ResourceCPU:    resource.MustParse("1000m"),
		},
	}

	result := normalizeResourceRequirements(input)

	// Check that 4096Mi becomes 4Gi and 1000m becomes 1
	expectedMemory := resource.MustParse("4Gi")
	expectedCPU := resource.MustParse("1")

	if !result.Limits[corev1.ResourceMemory].Equal(expectedMemory) {
		t.Errorf("Expected memory to be normalized to 4Gi, got %v", result.Limits[corev1.ResourceMemory])
	}
	if !result.Limits[corev1.ResourceCPU].Equal(expectedCPU) {
		t.Errorf("Expected CPU to be normalized to 1, got %v", result.Limits[corev1.ResourceCPU])
	}
}

func Test_generateConfigMap_deterministicOrder(t *testing.T) {
	redisInstance := &k8sv1alpha1.Redis{
		ObjectMeta: metav1.ObjectMeta{Name: "test-redis", Namespace: "default"},
		Spec: k8sv1alpha1.RedisSpec{
			Config: map[string]string{
				"save":      "60 1000",
				"maxmemory": "2gb",
				"timeout":   "0",
			},
		},
	}

	master := redis.Address{Host: "192.168.1.100", Port: "6379"}

	// Generate ConfigMap twice
	configMap1 := generateConfigMap(redisInstance, master)
	configMap2 := generateConfigMap(redisInstance, master)

	// Should be identical (deterministic)
	if configMap1.Data[configFileName] != configMap2.Data[configFileName] {
		t.Error("ConfigMap generation is not deterministic")
	}

	// Check that keys are in sorted order
	content := configMap1.Data[configFileName]
	if !strings.Contains(content, "maxmemory 2gb\nsave 60 1000\ntimeout 0") {
		t.Error("Config keys are not sorted alphabetically")
	}
}
