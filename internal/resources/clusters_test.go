// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"reflect"
	"sync"
	"testing"

	"k8s.io/apimachinery/pkg/types"

	"github.com/SlinkyProject/slurm-client/pkg/client"
	"github.com/SlinkyProject/slurm-client/pkg/client/fake"
)

func TestNewClusters(t *testing.T) {
	tests := []struct {
		name string
		want *Clusters
	}{
		{
			name: "Test new clutsers",
			want: &Clusters{
				clients: make(map[string]client.Client),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewClusters(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewClusters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClusters_Get(t *testing.T) {
	testClient := fake.NewFakeClient()
	c := make(map[string]client.Client)
	c["default/foo"] = testClient
	type fields struct {
		clients map[string]client.Client
	}
	type args struct {
		name types.NamespacedName
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   client.Client
	}{
		{
			name: "existing namespaced name",
			fields: fields{
				clients: c,
			},
			args: args{
				name: types.NamespacedName{
					Namespace: "default",
					Name:      "foo",
				},
			},
			want: testClient,
		},
		{
			name: "incorrect namespaced name",
			fields: fields{
				clients: c,
			},
			args: args{
				name: types.NamespacedName{
					Namespace: "default",
					Name:      "bar",
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Clusters{
				lock:    sync.RWMutex{},
				clients: tt.fields.clients,
			}
			if got := c.Get(tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Clusters.Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClusters_add(t *testing.T) {
	testClient := fake.NewFakeClient()
	c := make(map[string]client.Client)
	c["default/foo"] = testClient
	type fields struct {
		clients map[string]client.Client
	}
	type args struct {
		name   types.NamespacedName
		client client.Client
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Already has NamespacedName",
			fields: fields{
				clients: c,
			},
			args: args{
				name: types.NamespacedName{
					Name:      "foo",
					Namespace: "default",
				},
				client: testClient,
			},
			want: false,
		},
		{
			name: "Add a new NamespacedName",
			fields: fields{
				clients: c,
			},
			args: args{
				name: types.NamespacedName{
					Name:      "bar",
					Namespace: "default",
				},
				client: testClient,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Clusters{
				lock:    sync.RWMutex{},
				clients: tt.fields.clients,
			}
			if got := c.add(tt.args.name, tt.args.client); got != tt.want {
				t.Errorf("Clusters.add() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClusters_Add(t *testing.T) {
	testClient := fake.NewFakeClient()
	c := make(map[string]client.Client)
	c["default/foo"] = testClient
	type fields struct {
		clients map[string]client.Client
	}
	type args struct {
		name   types.NamespacedName
		client client.Client
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Already has NamespacedName",
			fields: fields{
				clients: c,
			},
			args: args{
				name: types.NamespacedName{
					Name:      "foo",
					Namespace: "default",
				},
				client: testClient,
			},
			want: true,
		},
		{
			name: "Add a new NamespacedName",
			fields: fields{
				clients: c,
			},
			args: args{
				name: types.NamespacedName{
					Name:      "bar",
					Namespace: "default",
				},
				client: testClient,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Clusters{
				lock:    sync.RWMutex{},
				clients: tt.fields.clients,
			}
			if got := c.Add(tt.args.name, tt.args.client); got != tt.want {
				t.Errorf("Clusters.Add() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClusters_Has(t *testing.T) {
	testClient := fake.NewFakeClient()
	c := make(map[string]client.Client)
	foo := types.NamespacedName{
		Namespace: "default",
		Name:      "foo",
	}
	bar := types.NamespacedName{
		Namespace: "default",
		Name:      "bar",
	}
	c["default/foo"] = testClient

	type fields struct {
		clients map[string]client.Client
	}
	type args struct {
		names []types.NamespacedName
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Does not have NamespacedName",
			fields: fields{
				clients: c,
			},
			args: args{
				names: append([]types.NamespacedName{}, bar),
			},
			want: false,
		},
		{
			name: "Has NamespacedName",
			fields: fields{
				clients: c,
			},
			args: args{
				names: append([]types.NamespacedName{}, bar, foo),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Clusters{
				lock:    sync.RWMutex{},
				clients: tt.fields.clients,
			}
			if got := c.Has(tt.args.names...); got != tt.want {
				t.Errorf("Clusters.Has() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClusters_Remove(t *testing.T) {
	testClient := fake.NewFakeClient()
	c := make(map[string]client.Client)
	c["default/foo"] = testClient
	type fields struct {
		clients map[string]client.Client
	}
	type args struct {
		name types.NamespacedName
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Remove client that exists",
			fields: fields{
				clients: c,
			},
			args: args{
				name: types.NamespacedName{
					Name:      "foo",
					Namespace: "default",
				},
			},
			want: true,
		},
		{
			name: "Remove client that does not exists",
			fields: fields{
				clients: c,
			},
			args: args{
				name: types.NamespacedName{
					Name:      "bar",
					Namespace: "default",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Clusters{
				lock:    sync.RWMutex{},
				clients: tt.fields.clients,
			}
			if got := c.Remove(tt.args.name); got != tt.want {
				t.Errorf("Clusters.Remove() = %v, want %v", got, tt.want)
			}
		})
	}
}
