package storage

import (
	"github.com/stretchr/testify/mock"
	"mattermost-voting-bot/internal/models"
)

type MockVoteStorage struct {
	mock.Mock
}

func (s *MockVoteStorage) CreateVote(vote *models.Vote) (uint32, []string, error) {
	args := s.Called(vote)
	return args.Get(0).(uint32), args.Get(1).([]string), args.Error(2)
}

func (s *MockVoteStorage) Vote(voteId uint32, userId string, answers []string) error {
	args := s.Called(voteId, userId, answers)
	return args.Error(0)
}

func (s *MockVoteStorage) GetResults(voteId uint32) (map[string]int, error) {
	args := s.Called(voteId)
	return args.Get(0).(map[string]int), args.Error(1)
}

func (s *MockVoteStorage) CloseVote(voteId uint32, userId string) error {
	args := s.Called(voteId, userId)
	return args.Error(0)
}

func (s *MockVoteStorage) DeleteVote(voteId uint32, userId string) error {
	args := s.Called(voteId, userId)
	return args.Error(0)
}

func (s *MockVoteStorage) GetVote(voteId uint32) ([]interface{}, error) {
	args := s.Called(voteId)
	return args.Get(0).([]interface{}), args.Error(0)
}

func (s *MockVoteStorage) AutoCloseExpiredVotes() error {
	args := s.Called()
	return args.Error(0)
}

func (s *MockVoteStorage) Close() error {
	args := s.Called()
	return args.Error(0)
}
