package websoketkit

//SubscriptionData SubscriptionData
type SubscriptionData struct {
	SessionID string `json:"sessionid"`
	Channel   string `json:"channel"`
	Data      string `json:"data"`
}

//SubscriptionData SubscriptionData
type SubscriptionRequest struct {
	Action  string `json:"action"`
	Channel string `json:"channel"`
}
