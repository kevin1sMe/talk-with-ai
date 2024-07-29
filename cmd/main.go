package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sashabaranov/go-openai"

	log "github.com/sirupsen/logrus"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"gitlab.mrlin.cc/kevinlin/ai-tell-you/internal/asr"
	myplayer "gitlab.mrlin.cc/kevinlin/ai-tell-you/internal/player"
	"gitlab.mrlin.cc/kevinlin/ai-tell-you/internal/recorder"
	"gitlab.mrlin.cc/kevinlin/ai-tell-you/internal/tts"
	"gitlab.mrlin.cc/kevinlin/ai-tell-you/internal/tui"
)

var (
	history = []openai.ChatCompletionMessage{}
)

var (
	apiKey = os.Getenv("OPENAI_API_KEY")
	// baseURL   = "https://api.openai.com/v1"
	baseURL = os.Getenv("BASE_URL")

	// tencent
	// appId     = 1255793008
	appId, _  = strconv.ParseInt(os.Getenv("TENCENTCLOUD_APP_ID"), 10, 64)
	secretId  = os.Getenv("TENCENTCLOUD_SECRET_ID")
	secretKey = os.Getenv("TENCENTCLOUD_SECRET_KEY")

	// default setting
	modelName       = "yi-large"
	voiceType       = int64(101016)
	emotionCategory = "neutral"
	speed           = float64(1)

	processing = false
)

func main() {
	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()

	// 清空文件内容
	if err := f.Truncate(0); err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}

	// 确保文件指针在文件开始位置
	if _, err := f.Seek(0, 0); err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true, // 启用完整时间戳
	})
	log.SetOutput(f)
	log.SetLevel(log.DebugLevel)

	if appId == 0 {
		log.Fatal("请设置TENCENTCLOUD_APP_ID环境变量")
	}

	if apiKey == "" {
		log.Fatal("请设置OPENAI_API_KEY环境变量")
	}

	if secretId == "" {
		log.Fatal("请设置TENCENTCLOUD_SECRET_ID环境变量")
	}

	if secretKey == "" {
		log.Fatal("请设置TENCENTCLOUD_SECRET_KEY环境变量")
	}

	// 使用你的OpenAI API密钥创建客户端
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = baseURL
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}

	client := openai.NewClientWithConfig(cfg)
	if client == nil {
		log.Fatal("client is nil")
	}

	credential := common.NewCredential(
		os.Getenv("TENCENTCLOUD_SECRET_ID"),
		os.Getenv("TENCENTCLOUD_SECRET_KEY"),
	)

	asrClient, err := asr.NewClient(credential)
	if err != nil {
		panic(err)
	}

	recorder := recorder.NewRecorder()

	// 创建和UI交互的事件通道
	eventChan := make(chan tui.Event, 1)
	inChan := make(chan tui.Event, 1)
	go func() {
		for e := range eventChan {
			log.Debug("recv event from main loop", e)
			switch e.Type {
			case "model":
				modelName = e.Payload
			case "tone":
				voiceType, _ = strconv.ParseInt(e.Payload, 10, 64)
			case "emotion":
				emotionCategory = e.Payload
			case "audio_start":
				log.Debug("main|收到录音开始事件...")
				recorder.Start()
			case "audio_stop":
				log.Debug("main|收到录音结束事件...")
				recorder.Stop()

				buf := recorder.Buffer()
				log.Debug("正在识别语音输入...")
				question := sendAudioToASR(asrClient, buf.Bytes())
				log.Debugf("识别到内容：%s", question)
				QA(client, question, inChan)
			case "question":
				log.Debug("main|收到输入问题事件...")
				QA(client, e.Payload, inChan)
			}
		}

		log.Fatal("main|事件通道已关闭")
	}()

	p := tea.NewProgram(tui.InitialModel(log.StandardLogger(), eventChan, inChan), tea.WithAltScreen(), tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		fmt.Printf("出错了: %v", err)
		return
	}

}

func QA(c *openai.Client, request string, inChan chan tui.Event) {
	// 等待进入非处理中时，才继续往下
	t := time.Tick(time.Second)
	for range t {
		if !processing {
			break
		} else {
			log.Debug("正在处理上一个操作，请稍后")
		}
	}

	defer func() {
		processing = false
		log.Debug("*** 本次处理完成 ***")
	}()

	// 构造新的用户提问, 并添加到历史记录中
	newMessage := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: request,
	}
	history = append(history, newMessage)

	// 创建流式请求
	textChan := make(chan string, 1000)
	wholeChan := make(chan string, 1)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Warn("CompletionStream goroutine start...")
		CompletionStream(c, modelName, history, textChan, wholeChan)
		log.Warn("✅CompletionStream goroutine exit")
	}()

	audioChan := make(chan []byte, 1000)
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Warn("StreamTTS goroutine start...")
		StreamTTS(voiceType, emotionCategory, textChan, audioChan)
		log.Warn("✅StreamTTS goroutine exit")
		close(audioChan)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Warn("PlayStreamAudio goroutine start...")
		PlayStreamAudio(audioChan)
		log.Warn("✅PlayStreamAudio goroutine exit")
	}()

	// 记录到历史中
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Debug("History goroutine start...")
		resp := <-wholeChan
		log.Debugf("resp: %s", resp)
		history = append(history, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: resp,
		})

		historyStr, _ := json.Marshal(history)
		inChan <- tui.Event{
			Type:    "history",
			Payload: string(historyStr),
		}
		log.Debug("✅History goroutine exit")
	}()

	wg.Wait()

	log.Debug("✅✅✅✅等待所有goroutine完成✅✅✅✅")
}

// CompletionStream 调用AI流式回答
func CompletionStream(client *openai.Client, model string, msgs []openai.ChatCompletionMessage, textChan chan string, wholeResp chan string) {
	log.Debug("正在向AI请教...")
	ctx := context.Background()

	respText := bytes.Buffer{}
	// 设置请求参数
	req := openai.ChatCompletionRequest{
		Model:     model,
		MaxTokens: 1000,
		Messages:  msgs,
		Stream:    true, // 启用流式传输
	}
	stream, err := client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return
	}
	defer stream.Close()

	// 处理流式响应
	log.Debug("Stream response: ")

	defer func() {
		log.Debugf("CompletionStream finished! wholeResp:%s", respText.String())
		wholeResp <- respText.String()
		log.Debug("fill wholeResp")
	}()

	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			close(textChan)
			log.Info("Stream processing completed, textChan closed") // 添加这行
			return
		}

		if err != nil {
			return
		}

		// log.Debugf("接收到AI流式响应： [%s]", resp.Choices[0].Delta.Content)
		content := resp.Choices[0].Delta.Content
		respText.Write([]byte(content))

		textChan <- content
	}
}

// StreamTTS 语音合成
// 读取textChan中的数据，将它以。分割，然后合成语音
func StreamTTS(voiceType int64, emotionCategory string, textChan chan string, audioChan chan []byte) {
	var buffer strings.Builder

	var wg sync.WaitGroup
	wg.Add(1)

	s := tts.NewRealTimeSpeechSynthesizer(int64(appId), secretId, secretKey, voiceType, emotionCategory, speed)

	sentenceChan := make(chan string)
	// 启动一个 goroutine 来处理语音转换， 这样才能按顺序
	go func() {
		defer wg.Done()
		index := 1
		for sentence := range sentenceChan {
			log.Debug("----------------------------------")
			log.Debugf("正在转换第[%d]段语音中，文字内容为:%s ", index, sentence)
			s.Run(sentence, audioChan)
			index++
			log.Debug("----------------------------------")
		}
		log.Info("**语音转换全部结束！！**")
	}()

	index := 1
	for {
		select {
		case resp, ok := <-textChan:
			if !ok {
				log.Debugf("TextChan closed, buf len:%d", buffer.Len())
				// Channel 已关闭
				if buffer.Len() > 0 {
					// 发送句子到通道
					sentenceChan <- strings.TrimSpace(buffer.String())
				}
				goto END
			}

			// log.Debugf("Speech recv [%q]", resp)
			buffer.WriteString(resp)

			// 按句号分割句子
			content := buffer.String()
			sentences := strings.Split(content, "。")

			// 重置 buffer
			buffer.Reset()

			for i, sentence := range sentences {
				sentence = strings.TrimSpace(sentence)
				if sentence == "" {
					continue
				}

				if i == len(sentences)-1 && !strings.HasSuffix(content, "。") {
					// 最后一个句子可能是不完整的，保存到 buffer 中
					buffer.WriteString(sentence)
				} else {
					sentenceChan <- sentence + "。"
					// log.Debugf("发送第[%d]句子到sentenceChan：%s", index, sentence)
					index++
				}
			}

		}
	}
END:
	close(sentenceChan)
	log.Debug("sentenceChan closed")

	wg.Wait()
}

// 语音识别
func sendAudioToASR(c *asr.ASRClient, data []byte) string {
	asrResult, err := c.ToVoice("wav", data)
	if err != nil {
		return err.Error()
	}
	// log.Debugf("voice to text: %v\n", asrResult)
	return asrResult
}

// 播放语音
func PlayStreamAudio(audioStream chan []byte) {
	log.Debug("正在准备播放语音...")
	player := myplayer.NewMyPlayer(audioStream)
	player.Reset()

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		player.Play()
		log.Debug("Player started")
	}()
	go func() {
		defer wg.Done()
		player.GracefulStop()
		log.Debug("GracefulStop finished")
	}()
	wg.Wait()
	log.Debug("语音播放完成，播放器退出...")
}
