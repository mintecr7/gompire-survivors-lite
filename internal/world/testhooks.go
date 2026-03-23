package world

import "horde-lab/internal/jobs"

func (w *World) TestOnlyDisableAIPool() {
	if w.aiPool != nil {
		w.aiPool.Close()
		w.aiPool = nil
	}
}

func (w *World) TestOnlySetAIReadyResult(res jobs.IntentResult) {
	if w.aiReadyResults == nil {
		w.aiReadyResults = make(map[uint64]jobs.IntentResult, 8)
	}
	w.aiReadyResults[res.Tick] = res
}

func (w *World) TestOnlySetAIPendingRequest(req jobs.IntentRequest) {
	if w.aiPendingRequests == nil {
		w.aiPendingRequests = make(map[uint64]jobs.IntentRequest, 8)
	}
	w.aiPendingRequests[req.Tick] = req
}
