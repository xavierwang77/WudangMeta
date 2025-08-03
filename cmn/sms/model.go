package sms

type JuheConfig struct {
	ApiUrl string
	Key    string
}

type TecentConfig struct {
	AppID      string
	AppKey     string
	SignName   string
	TemplateID string
}

type ShxTongConfig struct {
	ApiUrl   string
	UserName string
	Password string
	Template string
}
