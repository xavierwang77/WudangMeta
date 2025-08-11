package sms

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tecentSMS "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
	"go.uber.org/zap"
)

type Service interface {
	SendVerifyCode(phone string, code string) error
}

type juheServiceImpl struct {
}

type tecentServiceImpl struct {
}

type shxServiceImpl struct {
}

func NewService() Service {
	switch platform {
	case "juhe":
		return &juheServiceImpl{}
	case "tecent":
		return &tecentServiceImpl{}
	case "shx":
		return &shxServiceImpl{}
	default:
		z.Warn("sms platform is not supported", zap.String("platform", platform))
	}
	return nil
}

// SendVerifyCode 发送验证码
func (*juheServiceImpl) SendVerifyCode(phone string, code string) error {
	if juheConfig.Key == "" {
		z.Error("sms is not enabled")
		return fmt.Errorf("juhe sms key is empty")
	}
	if phone == "" || code == "" {
		z.Error("sms phone or code is empty")
		return fmt.Errorf("phone or code is empty")
	}
	if !IsValidPhone(phone) {
		z.Error("sms phone is invalid")
		return fmt.Errorf("phone is invalid")
	}

	// 初始化参数
	param := url.Values{}

	// 接口请求参数
	param.Set("mobile", phone)             // 接收短信的手机号码
	param.Set("tpl_id", "xxxx")            // 短信模板ID，请参考个人中心短信模板设置
	param.Set("tpl_value", "#code#=12341") // 模板变量，如无则不用填写
	param.Set("key", juheConfig.Key)       // 接口请求Key

	// 发送请求
	data, err := Post(juheConfig.ApiUrl, param)
	if err != nil {
		z.Error("sms is not enabled")
		return err
	} else {
		var netReturn map[string]interface{}
		jsonErr := json.Unmarshal(data, &netReturn)
		if jsonErr != nil {
			// 解析JSON异常，根据自身业务逻辑进行调整修改
			z.Error("sms is not enabled")
			return jsonErr
		} else {
			errorCode := netReturn["error_code"]
			reason := netReturn["reason"]
			data := netReturn["result"]

			if errorCode.(float64) == 0 {
				return nil
			} else {
				// 查询失败，根据自身业务逻辑进行调整修改
				z.Error("sms is not enabled")
				return fmt.Errorf("error_code: %v, reason: %v, data: %v", errorCode, reason, data)
			}
		}
	}
}

// SendVerifyCode 发送验证码
func (*tecentServiceImpl) SendVerifyCode(phone string, code string) error {
	if tecentConfig.AppKey == "" {
		z.Error("tecent sms is not enabled")
		return fmt.Errorf("tecent sms appKey is empty")
	}
	if phone == "" || code == "" {
		z.Error("sms phone or code is empty")
		return fmt.Errorf("phone or code is empty")
	}
	if !IsValidPhone(phone) {
		z.Error("sms phone is invalid")
		return fmt.Errorf("phone is invalid")
	}

	request := tecentSMS.NewSendSmsRequest()

	request.SmsSdkAppId = common.StringPtr(tecentConfig.AppID)
	request.SignName = common.StringPtr(tecentConfig.SignName)
	request.TemplateId = common.StringPtr(tecentConfig.TemplateID)

	request.TemplateParamSet = common.StringPtrs([]string{code, "5"})

	phoneNumber := "+86" + phone

	request.PhoneNumberSet = common.StringPtrs([]string{phoneNumber})
	request.SessionContext = common.StringPtr("")
	request.ExtendCode = common.StringPtr("")
	request.SenderId = common.StringPtr("")

	response, err := tecentClient.SendSms(request)
	if err != nil {
		z.Error("sms is not enabled", zap.Error(err))
		return err
	}

	b, _ := json.Marshal(response.Response)
	// 打印返回的json字符串
	fmt.Printf("%s", b)

	return nil
}

// SendVerifyCode 发送验证码
func (*shxServiceImpl) SendVerifyCode(phone string, code string) error {
	if shxConfig.ApiUrl == "" {
		z.Error("shx sms is not enabled")
		return fmt.Errorf("shx sms apiUrl is empty")
	}
	if phone == "" || code == "" {
		z.Error("sms phone or code is empty")
		return fmt.Errorf("phone or code is empty")
	}
	if !IsValidPhone(phone) {
		z.Error("sms phone is invalid")
		return fmt.Errorf("phone is invalid")
	}

	// 构造请求参数
	form := url.Values{}
	form.Set("UserName", shxConfig.UserName)
	form.Set("Password", shxConfig.Password)
	form.Set("TimeStamp", "")
	form.Set("MobileNumber", phone)
	form.Set("MsgContent", fmt.Sprintf(shxConfig.Template, code))
	form.Set("MsgIdentify", fmt.Sprintf("shx-%d", time.Now().UnixNano()))

	// 发送 POST 请求
	resp, err := http.PostForm(shxConfig.ApiUrl, form)
	if err != nil {
		z.Error("failed to send sms", zap.Error(err))
		return fmt.Errorf("failed to send sms: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			z.Error("failed to close body", zap.Error(err))
		}
	}(resp.Body)

	// 读取返回结果
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		z.Error("failed to read sms response", zap.Error(err))
		return fmt.Errorf("failed to read sms response: %w", err)
	}

	z.Info("sms sent", zap.String("response", string(body)))

	return nil
}
