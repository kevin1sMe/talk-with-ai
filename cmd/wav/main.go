package main

import (
	"fmt"
	"io"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/hajimehoshi/oto"
)

const bufferSize = 4096

func main() {
	file, err := os.Open("./samples/paimeng.wav")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	decoder := wav.NewDecoder(file)
	if !decoder.IsValidFile() {
		fmt.Println("Invalid WAV file")
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

	player := context.NewPlayer()
	defer player.Close()

	fmt.Println("Starting playback...")
	buffer := make([]byte, bufferSize*2) // 每个样本占用 2 字节
	for {
		pcmBuffer := &audio.IntBuffer{Data: make([]int, bufferSize), Format: format}
		n, err := decoder.PCMBuffer(pcmBuffer)
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
		if _, err := player.Write(buffer[:2*n]); err != nil {
			fmt.Println("Error writing to player:", err)
			return
		}
	}

	fmt.Println("Playback finished")
}

// 	buffer := make([]byte, bufferSize*2) // 每个样本占用 2 字节
// 	var wg sync.WaitGroup
// 	wg.Add(1)

// 	go func() {
// 		defer wg.Done()
// 		for {
// 			pcmBuffer := &audio.IntBuffer{Data: make([]int, bufferSize), Format: format}
// 			n, err := decoder.PCMBuffer(pcmBuffer)
// 			if err != nil {
// 				if err == io.EOF {
// 					break
// 				}
// 				fmt.Println("Error reading audio data:", err)
// 				return
// 			}

// 			// Convert int buffer to byte buffer
// 			for i := 0; i < n; i++ {
// 				val := int16(pcmBuffer.Data[i]) // 将 int 转换为 int16
// 				buffer[2*i] = byte(val & 0xFF)  // 低字节
// 				buffer[2*i+1] = byte(val >> 8)  // 高字节
// 			}

// 			if _, err := player.Write(buffer[:2*n]); err != nil {
// 				fmt.Println("Error writing to player:", err)
// 				return
// 			}
// 		}
// 	}()

// 	// 等待音频播放完成
// 	wg.Wait()

// 	fmt.Println("Playback finished")
// }
