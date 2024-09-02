package pb

import "fmt"

// Additional helpers on the generated types.

// Get the string encoding ob job ID and task index.
func (m *TaskMetadata) TaskID() string {
	return fmt.Sprintf("%s/%03d", m.GetJobID(), m.GetIndex())
}

// Fill any nil (!) task parameters from a parent task specification.
func (wt *WasiTaskArgs) InheritNil(parent *WasiTaskArgs) *WasiTaskArgs {
	if wt.Args == nil {
		wt.Args = parent.Args
	}
	if wt.Envs == nil {
		wt.Envs = parent.Envs
	}
	if wt.Stdin == nil {
		wt.Stdin = parent.Stdin
	}
	if wt.Rootfs == nil {
		wt.Rootfs = parent.Rootfs
	}
	if wt.Artifacts == nil {
		wt.Artifacts = parent.Artifacts
	}
	return wt
}
