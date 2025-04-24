// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"sync"

	"k8s.io/apimachinery/pkg/types"

	"github.com/SlinkyProject/slurm-client/pkg/client"
)

type Clusters struct {
	lock    sync.RWMutex
	clients map[string]client.Client
}

func NewClusters() *Clusters {
	return &Clusters{
		clients: make(map[string]client.Client),
	}
}

func (c *Clusters) Get(name types.NamespacedName) client.Client {
	c.lock.RLock()
	defer c.lock.RUnlock()
	client, ok := c.clients[name.String()]
	if ok {
		return client
	}
	return nil
}

func (c *Clusters) Has(names ...types.NamespacedName) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	for _, name := range names {
		if _, ok := c.clients[name.String()]; ok {
			return true
		}
	}
	return false
}

func (c *Clusters) add(name types.NamespacedName, client client.Client) bool {
	if _, ok := c.clients[name.String()]; !ok {
		ctx := context.TODO()
		go client.Start(ctx)
		c.clients[name.String()] = client
		return true
	}
	return false
}

func (c *Clusters) Add(name types.NamespacedName, client client.Client) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.remove(name)
	return c.add(name, client)
}

func (c *Clusters) remove(name types.NamespacedName) bool {
	if client, ok := c.clients[name.String()]; ok {
		client.Stop()
		delete(c.clients, name.String())
		return true
	}
	return false
}

func (c *Clusters) Remove(name types.NamespacedName) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.remove(name)
}
