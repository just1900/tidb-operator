// Copyright 2018 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"testing"

	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStatefulSetIsUpgrading(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name            string
		update          func(*apps.StatefulSet)
		expectUpgrading bool
	}

	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		set := &apps.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: metav1.NamespaceDefault,
			},
		}
		if test.update != nil {
			test.update(set)
		}
		b := StatefulSetIsUpgrading(set)
		if test.expectUpgrading {
			g.Expect(b).To(BeTrue())
		} else {
			g.Expect(b).To(BeFalse())
		}
	}
	tests := []*testcase{
		{
			name:            "ObservedGeneration is nil",
			update:          nil,
			expectUpgrading: false,
		},
		{
			name: "CurrentRevision not equal UpdateRevision",
			update: func(set *apps.StatefulSet) {
				set.Status.ObservedGeneration = 1000
				set.Status.CurrentRevision = "v1"
				set.Status.UpdateRevision = "v2"
			},
			expectUpgrading: true,
		},
		{
			name: "set.Generation > *set.Status.ObservedGeneration && *set.Spec.Replicas == set.Status.Replicas",
			update: func(set *apps.StatefulSet) {
				set.Generation = 1001
				set.Status.ObservedGeneration = 1000
				set.Status.CurrentRevision = "v1"
				set.Status.UpdateRevision = "v1"
				set.Status.Replicas = 3
				set.Spec.Replicas = func() *int32 { var i int32 = 3; return &i }()
			},
			expectUpgrading: true,
		},
		{
			name: "replicas not equal",
			update: func(set *apps.StatefulSet) {
				set.Generation = 1001
				set.Status.ObservedGeneration = 1000
				set.Status.CurrentRevision = "v1"
				set.Status.UpdateRevision = "v1"
				set.Status.Replicas = 3
				set.Spec.Replicas = func() *int32 { var i int32 = 2; return &i }()
			},
			expectUpgrading: false,
		},
	}

	for _, test := range tests {
		testFn(test, t)
	}
}

func TestNotExistMount(t *testing.T) {
	oldSTS := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testem",
		},
		Spec: apps.StatefulSetSpec{
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pvc1",
					},
				},
			},
		},
	}

	newSTS := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testem",
		},
		Spec: apps.StatefulSetSpec{
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "notexist",
					},
				},
			},
		},
	}
	newSTS.Spec.Template.Spec.Volumes = []corev1.Volume{
		{
			Name: "v1",
		},
	}

	g := NewGomegaWithT(t)
	mp := notExistMount(newSTS, oldSTS)
	g.Expect(mp).Should(BeEmpty())

	// test mount volume in oldSTS.Spec.VolumeClaimTemplates
	c := corev1.Container{VolumeMounts: []corev1.VolumeMount{
		{
			Name: "pvc1",
		},
	}}
	newSTS.Spec.Template.Spec.Containers = []corev1.Container{c}
	mp = notExistMount(newSTS, oldSTS)
	g.Expect(mp).Should(BeEmpty())

	// test mount volume in newSTS.Spec.Template.Spec.Volumes
	c = corev1.Container{VolumeMounts: []corev1.VolumeMount{
		{
			Name: "v1",
		},
	}}
	newSTS.Spec.Template.Spec.Containers = []corev1.Container{c}
	mp = notExistMount(newSTS, oldSTS)
	g.Expect(mp).Should(BeEmpty())

	// test mount volume in newSTS.Spec.Template.Spec.Volumes
	// but not in newSTS.Spec.Template.Spec.Volumes
	c = corev1.Container{VolumeMounts: []corev1.VolumeMount{
		{
			Name: "notexist",
		},
	}}
	newSTS.Spec.Template.Spec.Containers = []corev1.Container{c}
	mp = notExistMount(newSTS, oldSTS)
	g.Expect(mp).ShouldNot(BeEmpty())
}
