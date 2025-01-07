package values

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	slurmv1 "nebius.ai/slurm-operator/api/v1"
)

func TestBuildSlurmWorkerFrom(t *testing.T) {
	clusterName := "test-cluster"

	sharedMemorySizeValue := resource.NewQuantity(1, resource.DecimalSI)

	worker := &slurmv1.SlurmNodeWorker{
		Volumes: slurmv1.SlurmNodeWorkerVolumes{
			SharedMemorySize: sharedMemorySizeValue,
		},
	}
	ncclSettings := &slurmv1.NCCLSettings{}

	result := buildSlurmWorkerFrom(clusterName, worker, ncclSettings, false)

	if result.SlurmNode != *worker.SlurmNode.DeepCopy() {
		t.Errorf("Expected SlurmNode to be %v, but got %v", *worker.SlurmNode.DeepCopy(), result.SlurmNode)
	}
	if result.NCCLSettings != *ncclSettings.DeepCopy() {
		t.Errorf("Expected NCCLSettings to be %v, but got %v", *ncclSettings.DeepCopy(), result.NCCLSettings)
	}
	if result.SharedMemorySize != sharedMemorySizeValue {
		t.Errorf("Expected SharedMemorySize to be %v, but got %v", sharedMemorySizeValue, result.SharedMemorySize)
	}
}
