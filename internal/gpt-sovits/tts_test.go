package gptsovits

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	// url := "http://127.0.0.1:6006/tts"
	// 	requestBody := map[string]interface{}{
	// 		// "text":              "先帝创业未半而中道崩殂，今天下三分，益州疲弊，此诚危急存亡之秋也。",
	// 		// "text":              "闺中少妇未曾有过相思离别之愁，在明媚的春日，她精心装扮之后兴高采烈登上翠楼。 忽见野外杨柳青青春意浓，真后悔让丈夫从军边塞，建功封侯。   ",
	// 		"text":              "王之涣（688年—742年），是盛唐时期的著名诗人，字季凌，汉族，绛州（今山西新绛县）人。豪放不羁，常击剑悲歌，其诗多被当时乐工制曲歌唱。名动一时，他常与高适、王昌龄等相唱和，以善于描写边塞风光著称。其代表作有《登鹳雀楼》、《凉州词》等。“白日依山尽，黄河入海流。欲穷千里目，更上一层楼”，更是千古绝唱",
	// 		"text_lang":         "zh",
	// 		"ref_audio_path":    "/mnt/c/Users/kevinlin/Source/GPT-SoVITS/samples/paimeng.wav",
	// 		"prompt_lang":       "zh",
	// 		"prompt_text":       "哇，这个，还有这个…只是和史莱姆打了一场，就有这么多结论吗？",
	// 		"text_split_method": "cut5",
	// 		"batch_size":        1,
	// 		"media_type":        "wav",
	// 		"streaming_mode":    true,
	// 	}
	s := NewGPTSovits("http://127.0.0.1:6006/tts", RequestParams{
		TextLang:        "zh",
		RefAudioPath:    "/mnt/c/Users/kevinlin/Source/GPT-SoVITS/samples/paimeng.wav",
		PromptText:      "哇，这个，还有这个…只是和史莱姆打了一场，就有这么多结论吗？",
		PromptLang:      "zh",
		TextSplitMethod: "cut5",
		BatchSize:       1,
		MediaType:       "wav",
		StreamingMode:   true,
	})

	text := "先帝创业未半而中道崩殂，今天下三分，益州疲弊，此诚危急存亡之秋也。"
	audioStream := make(chan []byte)
	s.Run(text, audioStream)
	assert.NotZero(t, len(audioStream))
}
