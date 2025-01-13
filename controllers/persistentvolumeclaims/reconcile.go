/*
Copyright 2025 The KubeVirt CSI driver Operator Authors.

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

package persistentvolumeclaims

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *Reconciler) reconcilePVCs(ctx context.Context, pvc *corev1.PersistentVolumeClaim) error {
	if pvc.Status.Phase == corev1.ClaimBound {
		assignedNodeName := pvc.Annotations["volume.kubernetes.io/selected-node"]
		assignedNode := &corev1.Node{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: assignedNodeName}, assignedNode); err != nil {
			return err
		}

		zone := assignedNode.Labels["topology.kubernetes.io/zone"]
		region := assignedNode.Labels["topology.kubernetes.io/region"]

		pv := &corev1.PersistentVolume{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: pvc.Spec.VolumeName}, pv); err != nil {
			return err
		}

		var nodeSelectorTerms []corev1.NodeSelectorTerm
		if zone != "" {
			nodeSelectorTerms = append(nodeSelectorTerms, corev1.NodeSelectorTerm{
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "topology.kubernetes.io/zone",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{zone},
					},
				},
			})
		}

		if region != "" {
			nodeSelectorTerms = append(nodeSelectorTerms, corev1.NodeSelectorTerm{
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      "topology.kubernetes.io/region",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{region},
					},
				},
			})
		}

		if len(nodeSelectorTerms) > 0 {
			pv.Spec.NodeAffinity = &corev1.VolumeNodeAffinity{
				Required: &corev1.NodeSelector{
					NodeSelectorTerms: nodeSelectorTerms,
				},
			}
		}

		pv = pv.DeepCopy()
		if err := r.Client.Update(ctx, pv); err != nil {
			return err
		}
	}
	return nil
}
