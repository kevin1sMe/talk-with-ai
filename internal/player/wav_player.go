package myplayer

import (
	"bytes"
	"fmt"
	"io"
	"time"

	// "github.com/ebitengine/oto/v3"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/hajimehoshi/oto"
	log "github.com/sirupsen/logrus"
)

const bufferSize = 4096
const minWavDataSize = 50

type WavPlayer struct {
	buffer       *bytes.Buffer
	decoder      *wav.Decoder
	player       *oto.Player
	format       *audio.Format
	audioStream  <-chan []byte
	readFinished bool
}

func NewWavPlayer(audioStream <-chan []byte) *WavPlayer {
	log.Debug("正在初始化播放器")
	return &WavPlayer{
		buffer:       &bytes.Buffer{},
		audioStream:  audioStream,
		readFinished: false,
	}
}

func (p *WavPlayer) Play() {
	log.Debug("正在调用播放器来播放语音")
	go p.readFromStream()

	buffer := make([]byte, bufferSize*2) // 每个样本占用 2 字节
	for {
		if p.readFinished || p.buffer.Len() >= minWavDataSize {
			log.Debugf("已经收取足够语音数据，正在初始化解码器, readFinished:%v, buf len:%d", p.readFinished, p.buffer.Len())
			// 确保播放器在播放前初始化
			if p.player != nil {
				log.Debug("播放器已经存在，关闭现有播放器")
				p.player.Close()
			}

			p.initializePlayer()

			// if p.player != nil && !p.player.IsPlaying() {
			// 	log.Debug("未在播放中，调用播放器来播放语音, Play!")
			// 	time.Sleep(500 * time.Millisecond)
			// 	p.player.Play()
			// }

			// if p.player != nil && p.player.IsPlaying() {
			// 	log.Debug("播放器已经进入Playing")
			// 	break
			// }

			// log.Debug("未在播放中，将会重置播放器!")
			pcmBuffer := &audio.IntBuffer{Data: make([]int, bufferSize), Format: p.format}
			n, err := p.decoder.PCMBuffer(pcmBuffer)
			fmt.Println("Read", n, "bytes")
			if err != nil {
				if err == io.EOF {
					break
				}
				fmt.Println("Error reading audio data:", err)
				return
			}
			if n == 0 {
				break
			}

			// Convert int buffer to byte buffer
			for i := 0; i < n; i++ {
				val := int16(pcmBuffer.Data[i]) // 将 int 转换为 int16
				buffer[2*i] = byte(val & 0xFF)  // 低字节
				buffer[2*i+1] = byte(val >> 8)  // 高字节
			}

			fmt.Println("Writing to player, n=", n)
			if _, err := p.player.Write(buffer[:2*n]); err != nil {
				fmt.Println("Error writing to player:", err)
				return
			}
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func (p *WavPlayer) Reset() {
	p.buffer.Reset()
	p.readFinished = false
}

func (p *WavPlayer) readFromStream() {
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

// func (p *MyPlayer) monitorPlayback() {
// 	playStarted := false
// 	for {
// 		if p.player != nil {
// 			if p.player.IsPlaying() {
// 				playStarted = true
// 				log.Debug("已经进入播放状态：playStarted=true")
// 			} else if playStarted && p.buffer.Len() == 0 {
// 				log.Debugf("播放已经开始并结束，且语音数据已经收取完毕，将停止播放, 播放器缓存长度：%d", p.player.BufferedSize())
// 				return
// 			} else {
// 				log.Debugf("player已初始化，但未进入播放状态，我会尝试再次调用播放，等待结束中...剩余长度:%d, 播放器缓存长度:%d, err:%v", p.buffer.Len(), p.player.BufferedSize(), p.player.Err())
// 				p.player.Play()
// 			}
// 		} else {
// 			log.Warnf("player未初始化，播放器未在播放中! 语音数据长度:%d", p.buffer.Len())
// 		}
// 		time.Sleep(time.Second)
// 	}
// }

// func (p *MyPlayer) GracefulStop() {
// 	p.monitorPlayback()
// 	log.Debugf("播放正常结束")
// 	if p.player != nil {
// 		p.player.Close()
// 		p.player = nil
// 	}
// }

func (p *WavPlayer) initializePlayer() {
	log.Debug("正在初始化解码器")
	var err error
	// p.decoder, err = mp3.NewDecoder(p.buffer)
	// if err != nil {
	// 	log.Fatalf("mp3.NewDecoder failed: %v", err)
	// 	return
	// }

	decoder := wav.NewDecoder(bytes.NewReader(p.buffer.Bytes()))
	if !decoder.IsValidFile() {
		log.Fatalf("Invalid WAV file")
		return
	}

	format := decoder.Format()
	fmt.Println("Format:", format)
	context, err := oto.NewContext(format.SampleRate, format.NumChannels, 2, bufferSize)
	if err != nil {
		fmt.Println("Error creating oto context:", err)
		return
	}
	defer context.Close()

	p.format = format
	p.decoder = decoder
	p.player = context.NewPlayer()

	log.Debug("解码器初始化成功")
}
