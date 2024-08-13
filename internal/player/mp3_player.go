package myplayer

import (
	"bytes"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
	log "github.com/sirupsen/logrus"
)

const minDataSize = 200

var (
	otoContext *oto.Context
	once       sync.Once
)

// getOtoContext returns the singleton oto.Context
func getOtoContext() *oto.Context {
	once.Do(func() {
		var err error
		op := &oto.NewContextOptions{
			SampleRate:   16000, // Adjust based on your MP3 sample rate
			ChannelCount: 2,
			Format:       oto.FormatSignedInt16LE,
		}

		var readyChan chan struct{}
		otoContext, readyChan, err = oto.NewContext(op)
		if err != nil {
			log.Fatalf("oto.NewContext failed: %v", err)
		}
		<-readyChan
	})
	return otoContext
}

type MyPlayer struct {
	buffer       *bytes.Buffer
	decoder      *mp3.Decoder
	player       *oto.Player
	audioStream  <-chan []byte
	readFinished bool
}

func NewMyPlayer(audioStream <-chan []byte) *MyPlayer {
	log.Debug("正在初始化播放器")
	return &MyPlayer{
		buffer:       &bytes.Buffer{},
		audioStream:  audioStream,
		readFinished: false,
	}
}

func (p *MyPlayer) Play() {
	log.Debug("正在调用播放器来播放语音")
	go p.readFromStream()

	for {
		if p.readFinished || p.buffer.Len() >= minDataSize {
			log.Debugf("已经收取足够语音数据，正在初始化解码器, readFinished:%v, buf len:%d", p.readFinished, p.buffer.Len())
			// 确保播放器在播放前初始化
			if p.player != nil {
				log.Debug("播放器已经存在，关闭现有播放器")
				p.player.Close()
			}

			p.initializePlayer()

			if p.player != nil && !p.player.IsPlaying() {
				log.Debug("未在播放中，调用播放器来播放语音, Play!")
				time.Sleep(500 * time.Millisecond)
				p.player.Play()
			}

			if p.player != nil && p.player.IsPlaying() {
				log.Debug("播放器已经进入Playing")
				break
			}

			log.Debug("未在播放中，将会重置播放器!")
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func (p *MyPlayer) Reset() {
	p.buffer.Reset()
	p.readFinished = false
}

func (p *MyPlayer) readFromStream() {
	for {
		select {
		case data, ok := <-p.audioStream:
			if !ok {
				// Channel is closed, stop reading
				log.Warnf("audioStream closed? 将退出播放器")
				p.readFinished = true
				return
			}
			p.buffer.Write(data)
			log.Debugf("收取语音数据：%d, 剩余长度:%d", len(data), p.buffer.Len())
		}
	}
}

func (p *MyPlayer) monitorPlayback() {
	playStarted := false
	for {
		if p.player != nil {
			if p.player.IsPlaying() {
				playStarted = true
				log.Debug("已经进入播放状态：playStarted=true")
			} else if playStarted && p.buffer.Len() == 0 {
				log.Debugf("播放已经开始并结束，且语音数据已经收取完毕，将停止播放, 播放器缓存长度：%d", p.player.BufferedSize())
				return
			} else {
				log.Debugf("player已初始化，但未进入播放状态，我会尝试再次调用播放，等待结束中...剩余长度:%d, 播放器缓存长度:%d, err:%v", p.buffer.Len(), p.player.BufferedSize(), p.player.Err())
				p.player.Play()
			}
		} else {
			log.Warnf("player未初始化，播放器未在播放中! 语音数据长度:%d", p.buffer.Len())
		}
		time.Sleep(time.Second)
	}
}

func (p *MyPlayer) GracefulStop() {
	p.monitorPlayback()
	log.Debugf("播放正常结束")
	if p.player != nil {
		p.player.Close()
		p.player = nil
	}
}

func (p *MyPlayer) initializePlayer() {
	log.Debug("正在初始化解码器")
	var err error
	// p.decoder, err = mp3.NewDecoder(p.buffer)
	// if err != nil {
	// 	log.Fatalf("mp3.NewDecoder failed: %v", err)
	// 	return
	// }

	p.decoder, err = mp3.NewDecoder(p.buffer)
	if err != nil {
		log.Fatalf("mp3.NewDecoder failed: %v", err)
		return
	}

	otoCtx := getOtoContext()
	p.player = otoCtx.NewPlayer(p.decoder)
	if p.player == nil {
		log.Fatal("otoCtx.NewPlayer failed")
	}
	log.Debug("解码器初始化成功")
}
