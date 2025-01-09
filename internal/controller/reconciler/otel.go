package reconciler

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	slurmv1 "nebius.ai/slurm-operator/api/v1"
	"nebius.ai/slurm-operator/internal/logfield"

	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

type OtelReconciler struct {
	*Reconciler
}

var (
	_ patcher = &OtelReconciler{}
)

func NewOtelReconciler(r *Reconciler) *OtelReconciler {
	return &OtelReconciler{
		Reconciler: r,
	}
}

func (r *OtelReconciler) Reconcile(
	ctx context.Context,
	cluster *slurmv1.SlurmCluster,
	desired *otelv1beta1.OpenTelemetryCollector,
	deps ...metav1.Object,
) error {
	if desired == nil {
		// If desired is nil, delete the OpenTelemetryCollector
		// TODO: Using error or desired is nil presence as an indicator for resource deletion doesn't seem good
		// We should use conditions instead. if condition is met and resource exists, delete it
		// MSP-2715 - task to improve resource deletion
		log.FromContext(ctx).Info(fmt.Sprintf("Deleting OpenTelemetryCollector %s-collector, because of OpenTelemetryCollector is not needed", cluster.Name))
		return r.deleteIfOwnedByController(ctx, cluster)
	}
	if err := r.reconcile(ctx, cluster, desired, r.patch, deps...); err != nil {
		log.FromContext(ctx).
			WithValues(logfield.ResourceKV(desired)...).
			Error(err, "Failed to reconcile OpenTelemetryCollector")
		return errors.Wrap(err, "reconciling OpenTelemetryCollector")
	}
	return nil
}

func (r *OtelReconciler) deleteIfOwnedByController(
	ctx context.Context,
	cluster *slurmv1.SlurmCluster,
) error {
	otel, err := r.getOtel(ctx, cluster)
	if apierrors.IsNotFound(err) {
		log.FromContext(ctx).Info("Service not found, skipping deletion")
		return nil
	}

	if err != nil {
		return errors.Wrap(err, "getting OpenTelemetryCollector")
	}

	isOwner := isControllerOwnerOtel(otel, cluster)
	if !isOwner {
		log.FromContext(ctx).Info("OpenTelemetryCollector is not owned by the controller, skipping deletion")
		return nil
	}
	// The controller is the owner of the OpenTelemetryCollector, delete it
	return r.deleteOtelOwnedByController(ctx, cluster, otel)
}

func (r *OtelReconciler) getOtel(ctx context.Context, cluster *slurmv1.SlurmCluster) (*otelv1beta1.OpenTelemetryCollector, error) {
	otel := &otelv1beta1.OpenTelemetryCollector{}
	err := r.Get(
		ctx,
		types.NamespacedName{
			Namespace: cluster.Namespace,
			Name:      cluster.Name,
		},
		otel,
	)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// otel doesn't exist, nothing to do
			return otel, nil
		}
		// Other error occurred
		return nil, errors.Wrap(err, "getting Worker OpenTelemetryCollector")
	}
	return otel, nil
}

// Function to check if the controller is the owner
func isControllerOwnerOtel(otel *otelv1beta1.OpenTelemetryCollector, cluster *slurmv1.SlurmCluster) bool {
	// Check if the controller is the owner of the Role
	isOwner := false
	for _, ownerRef := range otel.GetOwnerReferences() {
		if ownerRef.Kind == slurmv1.SlurmClusterKind && ownerRef.Name == cluster.Name {
			isOwner = true
			break
		}
	}

	return isOwner
}

func (r *OtelReconciler) deleteOtelOwnedByController(
	ctx context.Context,
	cluster *slurmv1.SlurmCluster,
	otel *otelv1beta1.OpenTelemetryCollector,
) error {
	// Delete the Role
	err := r.Client.Delete(ctx, otel)
	if err != nil {
		log.FromContext(ctx).
			WithValues("cluster", cluster.Name).
			Error(err, "Failed to delete Worker OpenTelemetryCollector")
		return errors.Wrap(err, "deleting Worker OpenTelemetryCollector")
	}
	return nil
}

func (r *OtelReconciler) patch(existing, desired client.Object) (client.Patch, error) {
	patchImpl := func(dst, src *otelv1beta1.OpenTelemetryCollector) client.Patch {
		res := client.MergeFrom(dst.DeepCopy())
		dst.Spec = src.Spec
		return res
	}
	return patchImpl(existing.(*otelv1beta1.OpenTelemetryCollector), desired.(*otelv1beta1.OpenTelemetryCollector)), nil
}
