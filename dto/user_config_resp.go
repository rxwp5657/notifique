package dto

type UserConfigResp struct {
	Channel string `json:"channel"`
	OptedIn bool   `json:"opted-in"`
}
