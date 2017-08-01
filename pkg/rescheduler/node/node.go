/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package node

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
	corelisters "k8s.io/kubernetes/pkg/client/listers/core/v1"
)

// ReadyNodes returns ready nodes irrespective of whether they are
// schedulable or not.
func ReadyNodes(client clientset.Interface, stopChannel <-chan struct{}) ([]*v1.Node, error) {
	nl := getNodeLister(client, stopChannel)
	nodes, err := nl.List(labels.Everything())
	if err != nil {
		return []*v1.Node{}, err
	}

	if len(nodes) == 0 {
		var err error
		nItems, err := client.Core().Nodes().List(metav1.ListOptions{})
		if err != nil {
			return []*v1.Node{}, err
		}

		for _, node := range nItems.Items {
			nodes = append(nodes, &node)
		}
	}

	readyNodes := make([]*v1.Node, 0, len(nodes))
	for _, node := range nodes {
		if IsReady(node) {
			readyNodes = append(readyNodes, node)
		}
	}
	return readyNodes, nil
}

func getNodeLister(client clientset.Interface, stopChannel <-chan struct{}) corelisters.NodeLister {
	listWatcher := cache.NewListWatchFromClient(client.Core().RESTClient(), "nodes", v1.NamespaceAll, fields.Everything())
	store := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	nodeLister := corelisters.NewNodeLister(store)
	reflector := cache.NewReflector(listWatcher, &v1.Node{}, store, time.Hour)
	reflector.RunUntil(stopChannel)

	return nodeLister
}

func IsReady(node *v1.Node) bool {
	for i := range node.Status.Conditions {
		cond := &node.Status.Conditions[i]
		// We consider the node for scheduling only when its:
		// - NodeReady condition status is ConditionTrue,
		// - NodeOutOfDisk condition status is ConditionFalse,
		// - NodeNetworkUnavailable condition status is ConditionFalse.
		if cond.Type == v1.NodeReady && cond.Status != v1.ConditionTrue {
			fmt.Printf("Ignoring node %v with %v condition status %v", node.Name, cond.Type, cond.Status)
			return false
		} /*else if cond.Type == v1.NodeOutOfDisk && cond.Status != v1.ConditionFalse {
			glog.V(4).Infof("Ignoring node %v with %v condition status %v", node.Name, cond.Type, cond.Status)
			return false
		} else if cond.Type == v1.NodeNetworkUnavailable && cond.Status != v1.ConditionFalse {
			glog.V(4).Infof("Ignoring node %v with %v condition status %v", node.Name, cond.Type, cond.Status)
			return false
		}*/
	}
	// Ignore nodes that are marked unschedulable
	/*if node.Spec.Unschedulable {
		glog.V(4).Infof("Ignoring node %v since it is unschedulable", node.Name)
		return false
	}*/
	return true
}
