package wasimoffv1

// Additional helpers on the generated types.

// Fill any nil (!) task parameters from a parent task specification.
func (wt *Task_Wasip1_Params) InheritNil(parent *Task_Wasip1_Params) *Task_Wasip1_Params {
	if parent == nil {
		// nothing to do when parent is nil
		return wt
	}
	if wt.Binary == nil {
		wt.Binary = parent.Binary
	}
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

// Return a string list of needed files for a task request.
func (tr *Task_Request) GetRequiredFiles() (files []string) {
	files = make([]string, 0, 2) // usually max. binary + rootfs

	switch params := tr.Parameters.(type) {

	case *Task_Request_Wasip1:
		p := params.Wasip1
		if p.Binary != nil && p.Binary.GetRef() != "" {
			files = append(files, *p.Binary.Ref)
		}
		if p.Rootfs != nil && p.Rootfs.GetRef() != "" {
			files = append(files, *p.Rootfs.Ref)
		}

	case *Task_Request_Pyodide:
		// log.Fatalln("GetRequiredFiles is not implemented for Pyodide yet")

	}

	return files
}

// Check if the Result is OK or if it it the error type.
func (tr *Task_Response) OK() bool {
	_, ok := tr.Result.(*Task_Response_Error)
	// if the above assertion is ok, it *was* an error and this function should return false
	return !ok
}
