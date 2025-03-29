package main

import (
	"context"
	"fmt"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	mmplugin "github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattn/go-shellwords"
	"github.com/tarantool/go-tarantool/v2"
	"log"
	"mattermost-voting-bot/internal/models"
	"mattermost-voting-bot/internal/storage"
	"strconv"
	"strings"
	"time"
)

type VotePlugin struct {
	plugin.MattermostPlugin
	voteStorage storage.IVoteStorage
}

func NewVotePlugin() *VotePlugin {
	return &VotePlugin{}
}

func (p *VotePlugin) OnActivate() error {
	err := p.API.RegisterCommand(&model.Command{
		Trigger:          "vote",
		AutoComplete:     true,
		AutoCompleteDesc: "Vote plugin commands",
		AutoCompleteHint: "[action] [arguments]",
		Description:      "Vote plugin for creating and managing polls",
		DisplayName:      "Vote Plugin",
	})
	if err != nil {
		return fmt.Errorf("failed to register command: %w", err)
	}

	p.API.LogInfo("Vote command registered successfully")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	dialer := tarantool.NetDialer{
		Address: "tarantool:3301",
		User:    "guest",
	}
	opts := tarantool.Opts{
		Timeout: time.Second,
	}

	conn, err := tarantool.Connect(ctx, dialer, opts)
	if err != nil {
		return fmt.Errorf("connection refused: %w", err)
	}

	p.API.LogInfo("Connected to Tarantool successfully")

	p.voteStorage = storage.NewVoteStorage(conn)
	go p.autoCloseExpiredVotes()
	return nil
}

func (p *VotePlugin) OnDeactivate() error {
	if err := p.voteStorage.Close(); err != nil {
		log.Printf("Failed to close storage: %v", err)
		return err
	}
	return nil
}

func (p *VotePlugin) ExecuteCommand(_ *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	parsedArgs, err := shellwords.Parse(args.Command)
	if err != nil || len(parsedArgs) < 2 {
		return &model.CommandResponse{Text: "Invalid command format."}, nil
	}

	action := parsedArgs[1]

	switch action {
	case "create":
		question, answers, flags, err := parseCreateCommand(parsedArgs[2:])
		if err != nil {
			return &model.CommandResponse{Text: fmt.Sprintf("Error: %v", err)}, nil
		}

		isMultipleAnswers := flags["--multiple-answers"] == "true"
		isAnonymous := flags["--anonymous"] == "true"
		var expiresAt *time.Time
		if exp, ok := flags["--expires-at"]; ok {
			t, err := time.Parse(time.RFC3339, exp)
			if err != nil {
				return &model.CommandResponse{Text: "Invalid expiration time format."}, nil
			}
			if time.Now().After(t) {
				return &model.CommandResponse{Text: "Expiration time must be in the future."}, nil
			}
			expiresAt = &t
		}

		vote := models.NewVote(args.UserId, question, answers, isMultipleAnswers, isAnonymous, expiresAt)

		voteId, answerOptions, err := p.voteStorage.CreateVote(vote)
		if err != nil {
			return &model.CommandResponse{Text: fmt.Sprintf("Error creating vote: %v", err)}, nil
		}

		return &model.CommandResponse{
			Text: fmt.Sprintf("Vote created (ID: %d). Answers: %v", voteId, answerOptions),
		}, nil

	case "answer":
		if len(parsedArgs) < 4 {
			return &model.CommandResponse{Text: "Provide the answer to vote."}, nil
		}
		voteId, ok := parseUint32(parsedArgs[2])
		if !ok {
			return &model.CommandResponse{Text: "Provide the uint32 vote ID greater than zero to vote."}, nil
		}

		answers := parsedArgs[3:]
		err := p.voteStorage.Vote(voteId, args.UserId, answers)
		if err != nil {
			return &model.CommandResponse{Text: fmt.Sprintf("Error: %v", err)}, nil
		}

		return &model.CommandResponse{
			Text: fmt.Sprintf("Your answers for vote (%d): %v", voteId, answers),
		}, nil

	case "result":
		if len(parsedArgs) < 3 {
			return &model.CommandResponse{Text: "Provide the vote ID to get results."}, nil
		}
		voteId, ok := parseUint32(parsedArgs[2])
		if !ok {
			return &model.CommandResponse{Text: "Provide the uint32 vote ID greater than zero to get results."}, nil
		}

		results, err := p.voteStorage.GetResults(voteId)
		if err != nil {
			return &model.CommandResponse{Text: fmt.Sprintf("Error: %v", err)}, nil
		}

		resultText := "Results:\n"
		for answer, count := range results {
			resultText += fmt.Sprintf("%s: %d votes\n", answer, count)
		}

		return &model.CommandResponse{Text: resultText}, nil

	case "close":
		if len(parsedArgs) < 3 {
			return &model.CommandResponse{Text: "Provide the vote ID to close vote."}, nil
		}
		voteId, ok := parseUint32(parsedArgs[2])
		if !ok {
			return &model.CommandResponse{Text: "Provide the uint32 vote ID greater than zero to close vote."}, nil
		}

		err := p.voteStorage.CloseVote(voteId, args.UserId)
		if err != nil {
			return &model.CommandResponse{Text: fmt.Sprintf("Error closing vote: %v", err)}, nil
		}

		return &model.CommandResponse{Text: fmt.Sprintf("Vote %d closed.", voteId)}, nil

	case "delete":
		if len(parsedArgs) < 3 {
			return &model.CommandResponse{Text: "Provide the vote ID to delete vote."}, nil
		}
		voteId, ok := parseUint32(parsedArgs[2])
		if !ok {
			return &model.CommandResponse{Text: "Provide the uint32 vote ID greater than zero to delete vote."}, nil
		}

		err := p.voteStorage.DeleteVote(voteId, args.UserId)
		if err != nil {
			return &model.CommandResponse{Text: fmt.Sprintf("Error deleting vote: %v", err)}, nil
		}

		return &model.CommandResponse{Text: fmt.Sprintf("Vote %d deleted.", voteId)}, nil

	default:
		return &model.CommandResponse{Text: "Unknown action. Available: create, result, close, delete."}, nil
	}
}

func parseCreateCommand(parts []string) (string, []string, map[string]string, error) {
	if len(parts) < 2 {
		return "", nil, nil, fmt.Errorf("provide a question and at least one answer")
	}

	flags := make(map[string]string)
	var question string
	var answers []string

	for i := 0; i < len(parts); i++ {
		if strings.HasPrefix(parts[i], "--") {
			if i+1 < len(parts) && !strings.HasPrefix(parts[i+1], "--") {
				flags[parts[i]] = parts[i+1]
				i++
			} else {
				flags[parts[i]] = "true"
			}
		} else if question == "" {
			question = parts[i]
		} else {
			answers = append(answers, parts[i])
		}
	}

	if question == "" || len(answers) == 0 {
		return "", nil, nil, fmt.Errorf("invalid question or answers")
	}

	return question, answers, flags, nil
}

func parseUint32(str string) (uint32, bool) {
	num, err := strconv.ParseUint(str, 10, 32)
	if err != nil || num == 0 {
		return 0, false
	}
	return uint32(num), true
}

func (p *VotePlugin) autoCloseExpiredVotes() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := p.voteStorage.AutoCloseExpiredVotes()
			if err != nil {
				log.Printf("Error closing expired votes: %v", err)
			}
		}
	}
}

func main() {
	mmplugin.ClientMain(NewVotePlugin())
}
