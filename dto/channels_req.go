package dto

type ChannelsReq struct {
	Channels []string `json:"channels" bidning:"unique,dive,oneof=e-mai sms in-app"`
}
