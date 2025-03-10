// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build kubeapiserver && orchestrator

//nolint:revive // TODO(CAPP) Fix revive linter
package k8s

import (
	"k8s.io/apimachinery/pkg/labels"
	corev1Informers "k8s.io/client-go/informers/core/v1"
	corev1Listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/DataDog/datadog-agent/pkg/collector/corechecks/cluster/orchestrator/collectors"
	k8sProcessors "github.com/DataDog/datadog-agent/pkg/collector/corechecks/cluster/orchestrator/processors/k8s"
	"github.com/DataDog/datadog-agent/pkg/orchestrator"
)

// NewClusterCollectorVersions builds the group of collector versions.
func NewClusterCollectorVersions() collectors.CollectorVersions {
	return collectors.NewCollectorVersions(
		NewClusterCollector(),
	)
}

// ClusterCollector is a collector for Kubernetes clusters.
type ClusterCollector struct {
	informer  corev1Informers.NodeInformer
	lister    corev1Listers.NodeLister
	metadata  *collectors.CollectorMetadata
	processor *k8sProcessors.ClusterProcessor
}

// NewClusterCollector creates a new collector for the Kubernetes Cluster
// resource.
func NewClusterCollector() *ClusterCollector {
	return &ClusterCollector{
		metadata: &collectors.CollectorMetadata{
			IsDefaultVersion:                     true,
			IsStable:                             true,
			IsMetadataProducer:                   true,
			IsManifestProducer:                   true,
			SupportsManifestBuffering:            true,
			Name:                                 clusterName,
			NodeType:                             orchestrator.K8sCluster,
			SupportsTerminatedResourceCollection: false,
		},
		processor: k8sProcessors.NewClusterProcessor(),
	}
}

// Informer returns the shared informer.
func (c *ClusterCollector) Informer() cache.SharedInformer {
	return c.informer.Informer()
}

// Init is used to initialize the collector.
func (c *ClusterCollector) Init(rcfg *collectors.CollectorRunConfig) {
	c.informer = rcfg.OrchestratorInformerFactory.InformerFactory.Core().V1().Nodes()
	c.lister = c.informer.Lister()
}

// Metadata is used to access information about the collector.
func (c *ClusterCollector) Metadata() *collectors.CollectorMetadata {
	return c.metadata
}

// Run triggers the collection process.
func (c *ClusterCollector) Run(rcfg *collectors.CollectorRunConfig) (*collectors.CollectorRunResult, error) {
	list, err := c.lister.List(labels.Everything())
	if err != nil {
		return nil, collectors.NewListingError(err)
	}

	return c.Process(rcfg, list)
}

// Process is used to process the list of resources and return the result.
func (c *ClusterCollector) Process(rcfg *collectors.CollectorRunConfig, list interface{}) (*collectors.CollectorRunResult, error) {
	ctx := collectors.NewK8sProcessorContext(rcfg, c.metadata)

	processResult, processed, err := c.processor.Process(ctx, list)

	// This would happen when recovering from a processor panic. In the nominal
	// case we would have a positive integer set at the very end of processing.
	// If this is not the case then it means code execution stopped sooner.
	// Panic recovery will log more information about the error so we can figure
	// out the root cause.
	if processed == -1 {
		return nil, collectors.ErrProcessingPanic
	}

	// The cluster processor can return errors since it has to grab extra
	// information from the API server during processing.
	if err != nil {
		return nil, collectors.NewProcessingError(err)
	}

	result := &collectors.CollectorRunResult{
		Result:             processResult,
		ResourcesListed:    1,
		ResourcesProcessed: processed,
	}

	return result, nil
}
