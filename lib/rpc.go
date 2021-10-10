package lib

// SetUpTriggerResp is a response of set_up_trigger call to the remote RPC tracker driver.
type SetUpTriggerResp struct{}

// SetUpTriggerReq is a request to set up a trigger on the desired URL with specified variables.
type SetUpTriggerReq struct {
	URL  string
	Vars Vars
}

// Request describes a requests to tracker's action.
type Request struct {
	Method   string
	Vars     Vars
	TicketID string // might be empty, in case if task is not registered yet
}

// Response describes possible return values of the Interface.Call
type Response struct {
	ID string // id of the created task in the tracker.
}
