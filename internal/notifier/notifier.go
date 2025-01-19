package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type SlackNotifier struct {
	logger     *log.Logger
	webhookURL string
}

func NewSlackNotifier(webhookURL string) *SlackNotifier {
	return &SlackNotifier{
		logger:     log.New(log.Writer(), "[Slack] ", log.LstdFlags),
		webhookURL: webhookURL,
	}
}

type SlackMessage struct {
	Blocks []Block `json:"blocks"`
}

type Block struct {
	Type   string      `json:"type"`
	Text   *Text       `json:"text,omitempty"`
	Fields []FieldText `json:"fields,omitempty"`
}

type Text struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type FieldText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (s *SlackNotifier) NotifyNewTargets(targets []Target) error {
	if len(targets) == 0 {
		s.logger.Println("No new targets to notify")
		return nil
	}

	const maxBlocksPerMessage = 50

	message := SlackMessage{
		Blocks: []Block{
			{
				Type: "header",
				Text: &Text{
					Type: "plain_text",
					Text: "🎯 New Bug Bounty Targets Found",
				},
			},
		},
	}

	for i, target := range targets {
		targetBlock := Block{
			Type: "section",
			Fields: []FieldText{
				{
					Type: "mrkdwn",
					Text: fmt.Sprintf("*Program Name*\n%s", target.ProgramName),
				},
				{
					Type: "mrkdwn",
					Text: fmt.Sprintf("*Target*\n`%s`", target.Target),
				},
				{
					Type: "mrkdwn",
					Text: fmt.Sprintf("*Found*\n%s", target.FirstSeen.Format(time.RFC1123)),
				},
				{
					Type: "mrkdwn",
					Text: fmt.Sprintf("*Category*\n`%s`", target.Category),
				},
			},
		}
		message.Blocks = append(message.Blocks, targetBlock)

		if i < len(targets)-1 {
			message.Blocks = append(message.Blocks, Block{Type: "divider"})
		}
	}

	for i := 0; i < len(message.Blocks); i += maxBlocksPerMessage {
		end := i + maxBlocksPerMessage
		if end > len(message.Blocks) {
			end = len(message.Blocks)
		}

		chunk := SlackMessage{Blocks: message.Blocks[i:end]}

		payload, err := json.Marshal(chunk)
		if err != nil {
			return fmt.Errorf("failed to marshal slack message: %w", err)
		}

		// s.logger.Printf("Sending payload to Slack: %s", string(payload))
		resp, err := http.Post(s.webhookURL, "application/json", bytes.NewBuffer(payload))
		if err != nil {
			return fmt.Errorf("failed to send slack webhook: %w", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("slack webhook returned status code %d: %s", resp.StatusCode, string(body))
		}
	}

	s.logger.Printf("Successfully notified about %d new targets", len(targets))
	return nil
}
