package common_models

type ValidateOTP struct {
	OTP int    `json:"otp" bson:"otp"`
	ID  string `json:"_id" bson:"_id"`
}
