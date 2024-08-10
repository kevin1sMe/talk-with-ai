package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"
)

type RequestParams struct {
	TextLang        string `json:"text_lang"`
	RefAudioPath    string `json:"ref_audio_path"`
	PromptText      string `json:"prompt_text"`
	PromptLang      string `json:"prompt_lang"`
	TextSplitMethod string `json:"text_split_method"`
	BatchSize       int    `json:"batch_size"`
	MediaType       string `json:"media_type"`
	StreamingMode   bool   `json:"streaming_mode"`
}

type Sovits struct {
	URL    string
	Params RequestParams
}

func NewGPTSovits(url string, params RequestParams) {

	// 构建请求参数
	url := "http://127.0.0.1:6006/tts"
	requestBody := map[string]interface{}{
		// "text":              "先帝创业未半而中道崩殂，今天下三分，益州疲弊，此诚危急存亡之秋也。",
		// "text":              "闺中少妇未曾有过相思离别之愁，在明媚的春日，她精心装扮之后兴高采烈登上翠楼。 忽见野外杨柳青青春意浓，真后悔让丈夫从军边塞，建功封侯。   ",
		"text":              "王之涣（688年—742年），是盛唐时期的著名诗人，字季凌，汉族，绛州（今山西新绛县）人。豪放不羁，常击剑悲歌，其诗多被当时乐工制曲歌唱。名动一时，他常与高适、王昌龄等相唱和，以善于描写边塞风光著称。其代表作有《登鹳雀楼》、《凉州词》等。“白日依山尽，黄河入海流。欲穷千里目，更上一层楼”，更是千古绝唱",
		"text_lang":         "zh",
		"ref_audio_path":    "/mnt/c/Users/kevinlin/Source/GPT-SoVITS/samples/paimeng.wav",
		"prompt_lang":       "zh",
		"prompt_text":       "哇，这个，还有这个…只是和史莱姆打了一场，就有这么多结论吗？",
		"text_split_method": "cut5",
		"batch_size":        1,
		"media_type":        "wav",
		"streaming_mode":    true,
	}
	start := time.Now()

	// 将请求参数编码为 JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	// 发送 POST 请求
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	// 创建一个临时文件以保存接收到的数据
	tempFile, err := os.Create("output.wav") // 替换为适当的文件名
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer tempFile.Close()

	// 持续读取数据
	buffer := make([]byte, 1024) // 1KB 的缓冲区
	for {
		n, err := resp.Body.Read(buffer)
		if err == io.EOF {
			break // 数据读取完毕
		}
		if err != nil {
			fmt.Println("Error reading response body:", err)
			return
		}

		// 将读取的数据写入文件
		if _, err := tempFile.Write(buffer[:n]); err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}

		// 处理接收到的数据（例如：播放音频）
		// 这里可以添加播放音频的逻辑
	}

	fmt.Println("Audio stream has been saved to output.wav, cost time: ", time.Since(start))

	fmt.Println("Playing audio...", tempFile.Name())
	cmd := exec.Command("cmd.exe", "/C", "start", "wmplayer", tempFile.Name()) // 使用 Windows Media Player 播放音频
	err = cmd.Start()
	if err != nil {
		fmt.Println("Error starting playback:", err)
		return
	}

}
