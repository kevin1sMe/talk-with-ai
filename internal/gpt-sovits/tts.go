package gptsovits

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

type GPTSovits struct {
	URL    string
	Params RequestParams
}

func NewGPTSovits(url string, params RequestParams) *GPTSovits {
	return &GPTSovits{
		URL:    url,
		Params: params,
	}
}

func (s *GPTSovits) Run(text string, audioStream chan<- []byte) {
	// 构建请求参数
	requestBody := map[string]interface{}{
		// "text":              "先帝创业未半而中道崩殂，今天下三分，益州疲弊，此诚危急存亡之秋也。",
		// "text":              "闺中少妇未曾有过相思离别之愁，在明媚的春日，她精心装扮之后兴高采烈登上翠楼。 忽见野外杨柳青青春意浓，真后悔让丈夫从军边塞，建功封侯。   ",
		// "text":              "王之涣（688年—742年），是盛唐时期的著名诗人，字季凌，汉族，绛州（今山西新绛县）人。豪放不羁，常击剑悲歌，其诗多被当时乐工制曲歌唱。名动一时，他常与高适、王昌龄等相唱和，以善于描写边塞风光著称。其代表作有《登鹳雀楼》、《凉州词》等。“白日依山尽，黄河入海流。欲穷千里目，更上一层楼”，更是千古绝唱",
		"text":      text,
		"text_lang": "zh",
		// "ref_audio_path":    "/mnt/c/Users/kevinlin/Source/GPT-SoVITS/samples/paimeng.wav",
		"ref_audio_path": s.Params.RefAudioPath,
		"prompt_lang":    s.Params.PromptLang,
		// "prompt_text":       "哇，这个，还有这个…只是和史莱姆打了一场，就有这么多结论吗？",
		"prompt_text": s.Params.PromptText,
		// "text_split_method": "cut5",
		"text_split_method": s.Params.TextSplitMethod,
		// "batch_size":        1,
		"batch_size": s.Params.BatchSize,
		// "media_type":        "wav",
		"media_type": s.Params.MediaType,
		// "streaming_mode":    true,
		"streaming_mode": s.Params.StreamingMode,
	}
	start := time.Now()

	// 将请求参数编码为 JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	// 发送 POST 请求
	resp, err := http.Post(s.URL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	// 持续读取数据
	buffer := make([]byte, 1023) // 1KB 的缓冲区
	for {
		n, err := resp.Body.Read(buffer)
		if err == io.EOF {
			break // 数据读取完毕
		}
		if err != nil {
			fmt.Println("Error reading response body:", err)
			return
		}
		fmt.Println("Received data len", n)
		audioStream <- buffer[:n]
	}

	fmt.Println("cost time:", time.Since(start))
}
