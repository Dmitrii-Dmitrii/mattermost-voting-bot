package storage

import "mattermost-voting-bot/internal/models"

type IVoteStorage interface {
	CreateVote(vote *models.Vote) (uint32, []string, error)
	Vote(voteId uint32, votedId string, answers []string) error
	GetResults(voteId uint32) (map[string]int, error)
	CloseVote(voteId uint32, creatorId string) error
	DeleteVote(voteId uint32, creatorId string) error
	GetVote(voteId uint32) ([]interface{}, error)
	AutoCloseExpiredVotes() error
	Close() error
}
