package tts

import (
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	terror "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/regions"
	tts "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tts/v20190823"
)

type TTSClient struct {
	client      *tts.Client
	callbackURL string
	mu          sync.Mutex
	jobs        map[string]chan string // 异步任务队列, taskID -> resultURL
}

// NewClient 创建腾讯云TTS客户端， 用于将文本转为语音
func NewClient(c *common.Credential, cb string) (*TTSClient, error) {
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tts.tencentcloudapi.com"
	client, err := tts.NewClient(c, regions.Guangzhou, cpf)
	return &TTSClient{
		client:      client,
		callbackURL: cb,
		jobs:        make(map[string]chan string),
	}, err
}

// ToAudio 将文本转为语音返回，出错返回err
func (t *TTSClient) ToAudio(codec string, voiceType int64, text string) ([]byte, error) {
	request := tts.NewTextToVoiceRequest()

	request.Text = common.StringPtr(text)
	request.SessionId = common.StringPtr(string(uuid.New().String()))
	request.Codec = common.StringPtr(codec)
	request.VoiceType = common.Int64Ptr(voiceType)

	response, err := t.client.TextToVoice(request)
	if _, ok := err.(*terror.TencentCloudSDKError); ok {
		fmt.Printf("An API error has returned: %s", err)
		return []byte(""), err
	}
	if err != nil {
		panic(err)
	}

	b, err := base64.StdEncoding.DecodeString(*response.Response.Audio)
	if err != nil {
		panic(err)
	}
	return b, nil
}

// ToLongAudio 将长文本转为语音返回，出错返回err
// 返回 ： 语音文件的URL， 或错误
func (t *TTSClient) ToLongAudio(codec string, voiceType int64, text string) (string, error) {
	request := tts.NewCreateTtsTaskRequest()

	request.Text = common.StringPtr(text)
	request.Codec = common.StringPtr(codec)
	request.VoiceType = common.Int64Ptr(voiceType)
	request.ModelType = common.Int64Ptr(1)
	request.CallbackUrl = common.StringPtr(t.callbackURL)

	response, err := t.client.CreateTtsTask(request)
	if _, ok := err.(*terror.TencentCloudSDKError); ok {
		logrus.Warnf("An API error has returned: %s", err)
		return "", err
	}
	if err != nil {
		logrus.Warnf("An API error has returned: %s", err)
		return "", err
	}

	cbData := make(chan string)
	t.mu.Lock()
	t.jobs[*response.Response.Data.TaskId] = cbData
	t.mu.Unlock()

	logrus.Debugf("waiting for task %v callback", *response.Response.Data.TaskId)
	audioURL := <-cbData
	logrus.Debugf("recv task %v callback! aduioURL:%v", *response.Response.Data.TaskId, audioURL)

	return audioURL, nil
}

func (t *TTSClient) OnCallback(taskID string, resultURL string) {
	logrus.Debugf("recv task %v callback! aduioURL:%v", taskID, resultURL)
	if _, ok := t.jobs[taskID]; !ok {
		logrus.Warnf("taskID %v not found", taskID)
		return
	}

	t.jobs[taskID] <- resultURL
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.jobs, taskID)
}
