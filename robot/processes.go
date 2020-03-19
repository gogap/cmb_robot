package robot

import (
	"github.com/AllenDang/w32"
)

func findProcess(name string) uint32 {

	processIds := make([]uint32, 1024)

	var bytesReturned, cProcesses uint32 = 0, 0

	if !w32.EnumProcesses(processIds, 1024, &bytesReturned) {
		return 0
	}

	cProcesses = bytesReturned / 2

	for i := 0; i < int(cProcesses); i++ {
		if processIds[i] != 0 {
			if name == getProcName(processIds[i]) {
				return processIds[i]
			}
		}
	}

	return 0
}

func getProcName(pid uint32) string {

	hProcess := w32.OpenProcess(
		w32.PROCESS_VM_READ|w32.PROCESS_QUERY_INFORMATION,
		false,
		pid,
	)

	if hProcess == 0 {
		return ""
	}

	defer func() {
		w32.CloseHandle(hProcess)
	}()

	return w32.GetModuleBaseNameW(hProcess, 0)
}
