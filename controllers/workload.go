package controllers

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1beta1 "github.com/doodlescheduling/k8sdb-controller/api/v1beta1"
)

func downscaleWorkloads(ctx context.Context, c client.Client, workloads []infrav1beta1.WorkloadReference) ([]infrav1beta1.WorkloadReference, error) {
	for _, wl := range workloads {
		var r client.Object
		switch wl.Kind {
		case "Deployment":
			r = &appsv1.Deployment{}
		case "StatefulSet":
			r = &appsv1.StatefulSet{}
		case "ReplicaSet":
			r = &appsv1.ReplicaSet{}
		default:
			return workloads, fmt.Errorf("unknown workload kind %s given", wl.Kind)
		}

		err := c.Get(ctx, client.ObjectKey{
			Namespace: wl.Namespace,
			Name:      wl.Name,
		}, r)

		if err != nil {
			return workloads, fmt.Errorf("failed to lookup referencing workload: %w", err)
		}

		var zero int32
		switch wl.Kind {
		case "Deployment":
			if r.(*appsv1.Deployment).Spec.Replicas != &zero {
				wl.Replicas = r.(*appsv1.Deployment).Spec.Replicas
			}

			r.(*appsv1.Deployment).Spec.Replicas = &zero
		case "StatefulSet":
			if r.(*appsv1.StatefulSet).Spec.Replicas != &zero {
				wl.Replicas = r.(*appsv1.StatefulSet).Spec.Replicas
			}

			r.(*appsv1.StatefulSet).Spec.Replicas = &zero
		case "ReplicaSet":
			if r.(*appsv1.ReplicaSet).Spec.Replicas != &zero {
				wl.Replicas = r.(*appsv1.ReplicaSet).Spec.Replicas
			}

			r.(*appsv1.ReplicaSet).Spec.Replicas = &zero
		default:
			return workloads, fmt.Errorf("unknown workload kind %s given", wl.Kind)
		}

		err = c.Update(ctx, r)
		if err != nil {
			return workloads, fmt.Errorf("failed to scale down workload: %w", err)
		}
	}

	return workloads, nil
}

func upscaleWorkloads(ctx context.Context, c client.Client, workloads []infrav1beta1.WorkloadReference) error {
	for _, wl := range workloads {
		var r client.Object
		switch wl.Kind {
		case "Deployment":
			r = &appsv1.Deployment{}
		case "StatefulSet":
			r = &appsv1.StatefulSet{}
		case "ReplicaSet":
			r = &appsv1.ReplicaSet{}
		default:
			return fmt.Errorf("unknown workload kind %s given", wl.Kind)
		}

		err := c.Get(ctx, client.ObjectKey{
			Namespace: wl.Namespace,
			Name:      wl.Name,
		}, r)

		if err != nil {
			return fmt.Errorf("failed to lookup referencing workload: %w", err)
		}

		switch wl.Kind {
		case "Deployment":
			r.(*appsv1.Deployment).Spec.Replicas = wl.Replicas
		case "StatefulSet":
			r.(*appsv1.StatefulSet).Spec.Replicas = wl.Replicas
		case "ReplicaSet":
			r.(*appsv1.ReplicaSet).Spec.Replicas = wl.Replicas
		default:
			return fmt.Errorf("unknown workload kind %s given", wl.Kind)
		}

		err = c.Update(ctx, r)
		if err != nil {
			return fmt.Errorf("failed to scale up workload: %w", err)
		}
	}

	return nil
}
