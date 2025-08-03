package sms

import "testing"

func TestSendVerifyCode(t *testing.T) {
	service := NewService()

	err := service.SendVerifyCode("15819888226", "1234")
	if err != nil {
		t.Errorf("SendVerifyCode failed: %v", err)
	} else {
		t.Log("SendVerifyCode success")
	}
}
