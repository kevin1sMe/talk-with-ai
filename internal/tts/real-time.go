package tts

import (
	"sync"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/tencentcloud/tencentcloud-speech-sdk-go/common"
	"github.com/tencentcloud/tencentcloud-speech-sdk-go/tts"
)

// RealTimeSpeechSynthesizer 实时语音合成。根据所需要的音色、情感、速度等生成。流式传输
type RealTimeSpeechSynthesizer struct {
	SessionId string

	audioStream chan<- []byte
	total       int // 最长度，没啥用，打个日志

	appId      int64
	credential *common.Credential

	voiceType       int64   // 音色
	emotionCategory string  // 情绪
	speed           float64 // 语速. 1.5倍=2
}

func NewRealTimeSpeechSynthesizer(appId int64, secretId, secretKey string, voiceType int64, emotionCategory string, speed float64) *RealTimeSpeechSynthesizer {
	l := &RealTimeSpeechSynthesizer{
		appId:           appId,
		credential:      common.NewCredential(secretId, secretKey),
		voiceType:       voiceType,
		emotionCategory: emotionCategory,
		speed:           speed,
	}
	return l
}

// 返回SessionId
func (l *RealTimeSpeechSynthesizer) SessionID() string {
	return l.SessionId
}

func (l *RealTimeSpeechSynthesizer) OnSynthesisStart(r *tts.SpeechWsSynthesisResponse) {
	log.Debug("OnSynthesisStart")
}

func (l *RealTimeSpeechSynthesizer) OnSynthesisEnd(r *tts.SpeechWsSynthesisResponse) {
	// log.Debugf("OnSynthesisEnd,sessionId:%s response: %s", l.SessionId, r.ToString())
	log.Debug("OnSynthesisEnd")
}

func (l *RealTimeSpeechSynthesizer) OnAudioResult(data []byte) {
	l.audioStream <- data
	l.total += len(data)
	// log.Debugf("OnAudioResult, len(data):%d total:%d\n", len(data), l.total)
}

func (l *RealTimeSpeechSynthesizer) OnTextResult(r *tts.SpeechWsSynthesisResponse) {
	// log.Debugf("OnTextResult,sessionId:%s", l.SessionId)
}
func (l *RealTimeSpeechSynthesizer) OnSynthesisFail(r *tts.SpeechWsSynthesisResponse, err error) {
	log.Fatalf("OnSynthesisFail,sessionId:%s response: %s err:%s", l.SessionId, r.ToString(), err.Error())
}

func (l *RealTimeSpeechSynthesizer) Reset() {
	l.SessionId = uuid.New().String()
	l.total = 0
}
func (l *RealTimeSpeechSynthesizer) Run(text string, audioStream chan<- []byte) {
	log.Debug("开始转换语音: ", text, " voiceType:", l.voiceType, " emotionCategory:", l.emotionCategory)
	var wg sync.WaitGroup

	l.Reset()
	l.audioStream = audioStream

	wg.Add(1)
	go func() {
		defer wg.Done()
		synthesizer := tts.NewSpeechWsSynthesizer(l.appId, l.credential, l)
		synthesizer.SessionId = l.SessionID()
		synthesizer.VoiceType = l.voiceType
		synthesizer.Codec = "mp3"
		synthesizer.Text = text
		synthesizer.EnableSubtitle = true
		synthesizer.Speed = l.speed // 1.5x
		synthesizer.EmotionCategory = l.emotionCategory
		synthesizer.EmotionIntensity = 200
		//synthesizer.Debug = true
		//synthesizer.DebugFunc = func(message string) { log.Debug(message) }
		err := synthesizer.Synthesis()
		if err != nil {
			log.Fatal("语音合成失败", err)
			return
		}
		synthesizer.Wait()
		log.Debug("synthesizer completed")
	}()
	wg.Wait()

	// log.Debug("转换语音结束: ", text)
}
