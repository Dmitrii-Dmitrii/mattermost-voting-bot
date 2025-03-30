package models

type VoteTarantoolModel struct {
	Id                interface{}
	CreatorId         interface{}
	Question          interface{}
	Answers           interface{}
	Votes             interface{}
	IsActive          interface{}
	IsMultipleAnswers interface{}
	IsAnonymous       interface{}
	ExpiresAt         interface{}
}

func NewVoteTarantoolModel(id, creatorId, question, answers, votes, isActive, isMultipleAnswers, isAnonymous, expiresAt interface{}) *VoteTarantoolModel {
	return &VoteTarantoolModel{
		Id:                id,
		CreatorId:         creatorId,
		Question:          question,
		Answers:           answers,
		Votes:             votes,
		IsActive:          isActive,
		IsMultipleAnswers: isMultipleAnswers,
		IsAnonymous:       isAnonymous,
		ExpiresAt:         expiresAt,
	}
}
