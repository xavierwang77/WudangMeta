package sms

import (
	"fmt"

	"github.com/spf13/viper"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tecentSMS "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
	"go.uber.org/zap"
)

// 初始化聚合平台配置
func initJuheConfig() error {
	// 初始化配置信息
	juheConfig.ApiUrl = viper.GetString("sms.data.apiUrl")
	if juheConfig.ApiUrl == "" {
		z.Error("juhe apiUrl is empty")
		return fmt.Errorf("juhe apiUrl is empty")
	}
	juheConfig.Key = viper.GetString("sms.data.key")
	if juheConfig.Key == "" {
		z.Error("juhe key is empty")
		return fmt.Errorf("juhe key is empty")
	}
	return nil
}

// 初始化腾讯云平台配置
func initTecentConfig() error {
	// 初始化配置信息
	tecentConfig.AppID = viper.GetString("sms.data.appId")
	if tecentConfig.AppID == "" {
		z.Error("tecent appId is empty")
		return fmt.Errorf("tecent appId is empty")
	}
	tecentConfig.AppKey = viper.GetString("sms.data.appKey")
	if tecentConfig.AppKey == "" {
		z.Error("tecent appKey is empty")
		return fmt.Errorf("tecent appKey is empty")
	}
	tecentConfig.TemplateID = viper.GetString("sms.data.templateId")
	if tecentConfig.TemplateID == "" {
		z.Error("tecent templateId is empty")
		return fmt.Errorf("tecent templateId is empty")
	}
	tecentConfig.SignName = viper.GetString("sms.data.signName")
	if tecentConfig.SignName == "" {
		z.Error("tecent signName is empty")
		return fmt.Errorf("tecent signName is empty")
	}

	secretID := viper.GetString("sms.data.secretId")
	if secretID == "" {
		z.Error("tecent secretId is empty")
		return fmt.Errorf("tecent secretId is empty")
	}
	secretKey := viper.GetString("sms.data.secretKey")
	if secretKey == "" {
		z.Error("tecent secretKey is empty")
		return fmt.Errorf("tecent secretKey is empty")
	}

	var err error

	// 初始化客户端
	credential := common.NewCredential(
		secretID,
		secretKey,
	)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.ReqMethod = "POST"
	cpf.HttpProfile.ReqTimeout = 10 // 请求超时时间，单位为秒(默认60秒)
	cpf.HttpProfile.Endpoint = "sms.tencentcloudapi.com"
	cpf.SignMethod = "HmacSHA1"
	tecentClient, err = tecentSMS.NewClient(credential, "ap-guangzhou", cpf)
	if err != nil {
		z.Error("init tecent sms client failed", zap.Error(err))
		return fmt.Errorf("init tecent sms client failed: %v", err)
	}

	return nil
}

// 初始化闪信通平台配置
func initShxTongConfig() error {
	shxConfig.ApiUrl = viper.GetString("sms.data.apiUrl")
	if shxConfig.ApiUrl == "" {
		z.Error("shxtong apiUrl is empty")
		return fmt.Errorf("shxtong apiUrl is empty")
	}
	shxConfig.UserName = viper.GetString("sms.data.userName")
	if shxConfig.UserName == "" {
		z.Error("shxtong userName is empty")
		return fmt.Errorf("shxtong userName is empty")
	}
	shxConfig.Password = viper.GetString("sms.data.password")
	if shxConfig.Password == "" {
		z.Error("shxtong password is empty")
		return fmt.Errorf("shxtong password is empty")
	}
	shxConfig.Template = viper.GetString("sms.data.template")
	if shxConfig.Template == "" {
		z.Error("shxtong template is empty")
		return fmt.Errorf("shxtong template is empty")
	}
	return nil
}
