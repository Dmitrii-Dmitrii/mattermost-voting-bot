package main

import (
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"mattermost-voting-bot/internal/storage"
	"testing"
)

func TestHandleInvalidCommand(t *testing.T) {
	mockStorage := new(storage.MockVoteStorage)
	plugin := &VotePlugin{voteStorage: mockStorage}

	args := &model.CommandArgs{
		UserId:  "user123",
		Command: `/vote`,
	}

	response, err := plugin.ExecuteCommand(nil, args)

	assert.Nil(t, err)
	assert.Contains(t, response.Text, "Invalid command format.")
}

func TestExecuteCreateCommand(t *testing.T) {
	mockStorage := new(storage.MockVoteStorage)
	votePlugin := &VotePlugin{voteStorage: mockStorage}

	mockStorage.On("CreateVote", mock.Anything).Return(uint32(1), []string{"Go", "Java"}, nil)

	args := &model.CommandArgs{
		UserId:  "user123",
		Command: `/vote create "What is your favorite language?" "Go" "Java"`,
	}

	response, err := votePlugin.ExecuteCommand(nil, args)

	assert.Nil(t, err)
	assert.Contains(t, response.Text, "Vote created (ID: 1)")
}

func TestExecuteCreateCommandWithoutAnswers(t *testing.T) {
	mockStorage := new(storage.MockVoteStorage)
	plugin := &VotePlugin{voteStorage: mockStorage}

	args := &model.CommandArgs{
		UserId:  "user123",
		Command: `/vote create "What is your favorite language?"`,
	}

	response, err := plugin.ExecuteCommand(nil, args)

	assert.Nil(t, err)
	assert.Contains(t, response.Text, "Error: provide a question and at least one answer")
}

func TestExecuteCreateCommandWithInvalidFlags(t *testing.T) {
	mockStorage := new(storage.MockVoteStorage)
	plugin := &VotePlugin{voteStorage: mockStorage}

	mockStorage.On("CreateVote", mock.Anything).Return(uint32(1), []string{"Go", "Java"}, nil)

	args := &model.CommandArgs{
		UserId:  "user123",
		Command: `/vote create "What is your favorite language?" "Go" "Java" --ano`,
	}

	response, err := plugin.ExecuteCommand(nil, args)

	assert.Nil(t, err)
	assert.Contains(t, response.Text, "Vote created (ID: 1)")
}

func TestExecuteCreateCommandWithFlagExpiresAt(t *testing.T) {
	mockStorage := new(storage.MockVoteStorage)
	plugin := &VotePlugin{voteStorage: mockStorage}

	mockStorage.On("CreateVote", mock.Anything).Return(uint32(1), []string{"Go", "Java"}, nil)

	args := &model.CommandArgs{
		UserId:  "user123",
		Command: `/vote create "What is your favorite language?" "Go" "Java" --expires-at "2125-01-01T12:00:00Z"`,
	}

	response, err := plugin.ExecuteCommand(nil, args)

	assert.Nil(t, err)
	assert.Contains(t, response.Text, "Vote created (ID: 1)")
}

func TestExecuteCreateCommandWithInvalidFlagExpiresAt(t *testing.T) {
	mockStorage := new(storage.MockVoteStorage)
	plugin := &VotePlugin{voteStorage: mockStorage}

	mockStorage.On("CreateVote", mock.Anything).Return(uint32(1), []string{"Go", "Java"}, nil)

	args := &model.CommandArgs{
		UserId:  "user123",
		Command: `/vote create "What is your favorite language?" "Go" "Java" --expiresAt "2025-04-01"`,
	}

	response, err := plugin.ExecuteCommand(nil, args)

	assert.Nil(t, err)
	assert.Contains(t, response.Text, "Vote created (ID: 1)")
}

func TestExecuteAnswerCommand(t *testing.T) {
	mockStorage := new(storage.MockVoteStorage)
	plugin := &VotePlugin{voteStorage: mockStorage}

	mockStorage.On("Vote", uint32(123), "user123", []string{"Go"}).Return(nil)

	args := &model.CommandArgs{
		UserId:  "user123",
		Command: `/vote answer 123 "Go"`,
	}

	response, err := plugin.ExecuteCommand(nil, args)

	assert.Nil(t, err)
	assert.Contains(t, response.Text, "Your answers for vote (123): [Go]")
}

func TestExecuteCommandAnswerWithoutAnswers(t *testing.T) {
	mockStorage := new(storage.MockVoteStorage)
	plugin := &VotePlugin{voteStorage: mockStorage}

	mockStorage.On("Vote", uint32(123), "user123", []string{"Go"}).Return(nil)

	args := &model.CommandArgs{
		UserId:  "user123",
		Command: `/vote answer 123`,
	}

	response, err := plugin.ExecuteCommand(nil, args)

	assert.Nil(t, err)
	assert.Contains(t, response.Text, "Provide the answer to vote.")
}

func TestExecuteResultCommand(t *testing.T) {
	mockStorage := new(storage.MockVoteStorage)
	plugin := &VotePlugin{voteStorage: mockStorage}

	mockStorage.On("GetResults", uint32(123)).Return(map[string]int{"Go": 10, "Java": 5}, nil)

	args := &model.CommandArgs{
		UserId:  "user123",
		Command: `/vote result 123`,
	}

	response, err := plugin.ExecuteCommand(nil, args)

	assert.Nil(t, err)
	assert.Contains(t, response.Text, "Results:\nGo: 10 votes\nJava: 5 votes")
}

func TestExecuteCloseCommand(t *testing.T) {
	mockStorage := new(storage.MockVoteStorage)
	plugin := &VotePlugin{voteStorage: mockStorage}

	mockStorage.On("CloseVote", uint32(123), "user123").Return(nil)

	args := &model.CommandArgs{
		UserId:  "user123",
		Command: `/vote close 123`,
	}

	response, err := plugin.ExecuteCommand(nil, args)

	assert.Nil(t, err)
	assert.Contains(t, response.Text, "Vote 123 closed.")
}

func TestExecuteDeleteCommand(t *testing.T) {
	mockStorage := new(storage.MockVoteStorage)
	plugin := &VotePlugin{voteStorage: mockStorage}

	mockStorage.On("DeleteVote", uint32(123), "user123").Return(nil)

	args := &model.CommandArgs{
		UserId:  "user123",
		Command: `/vote delete 123`,
	}

	response, err := plugin.ExecuteCommand(nil, args)

	assert.Nil(t, err)
	assert.Contains(t, response.Text, "Vote 123 deleted.")
}
