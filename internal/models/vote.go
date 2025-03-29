package models

import (
	"math/rand"
	"time"
)

type Vote struct {
	Id                uint32              `json:"id"`
	CreatorId         string              `json:"creator_id"`
	Question          string              `json:"question"`
	Answers           []string            `json:"answers"`
	Votes             map[string][]string `json:"votes"`
	IsActive          bool                `json:"is_active"`
	IsMultipleAnswers bool                `json:"multiple_answers"`
	IsAnonymous       bool                `json:"is_anonymous"`
	ExpiresAt         *time.Time          `json:"expires_at,omitempty"`
}

func NewVote(creatorId, question string, answers []string, isMultipleAnswers, isAnonymous bool, expiresAt *time.Time) *Vote {
	return &Vote{
		Id:                generateId(),
		CreatorId:         creatorId,
		Question:          question,
		Answers:           answers,
		Votes:             make(map[string][]string),
		IsActive:          true,
		IsMultipleAnswers: isMultipleAnswers,
		IsAnonymous:       isAnonymous,
		ExpiresAt:         expiresAt,
	}
}

func generateId() uint32 {
	num := rand.Uint32()

	for num == 0 {
		num = rand.Uint32()
	}

	return num
}
