package asr

import (
	"encoding/base64"
	"fmt"

	asr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/asr/v20190614"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

type ASRClient struct {
	client *asr.Client
}

// NewClient 创建腾讯云ASR客户端， 用于将语音转为文本
func NewClient(c *common.Credential) (*ASRClient, error) {
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "asr.tencentcloudapi.com"
	client, err := asr.NewClient(c, "", cpf)

	return &ASRClient{
		client: client,
	}, err
}

// 将音频内容转为文本返回，出错返回err
func (a *ASRClient) ToVoice(fileType string, fileContents []byte) (string, error) {
	request := asr.NewSentenceRecognitionRequest()

	// 设置上传本地音频文件
	request.SourceType = common.Uint64Ptr(1)
	request.VoiceFormat = common.StringPtr(fileType)
	request.EngSerViceType = common.StringPtr("16k_zh")

	// 将buf的内容base64编码后设置给request.Data
	d64 := base64.StdEncoding.EncodeToString(fileContents)
	request.Data = common.StringPtr(d64)
	request.DataLen = common.Int64Ptr(int64(len(d64)))

	response, err := a.client.SentenceRecognition(request)
	if err != nil {
		return "", fmt.Errorf("fileType:%v, len:%v, err:%w", fileType, len(fileContents), err)
	}

	return *response.Response.Result, nil
}
