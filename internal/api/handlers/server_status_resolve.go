package handlers

func resolveEffectiveStatus(dbStatus, runtimeStatus string, cacheStatus *string) string {
	status := dbStatus
	runtime := runtimeStatus
	if runtime == "running" && status != "running" {
		status = "running"
	}
	if cacheStatus == nil {
		return status
	}
	live := *cacheStatus
	if live == "stopped" && (runtime == "running" || status == "running") {
		return "running"
	}
	return live
}
