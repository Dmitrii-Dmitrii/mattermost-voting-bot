package storage

import (
	"errors"
	"fmt"
	"github.com/tarantool/go-tarantool/v2"
	"mattermost-voting-bot/internal/models"
	"time"
)

type VoteStorage struct {
	conn *tarantool.Connection
}

func NewVoteStorage(conn *tarantool.Connection) *VoteStorage {
	return &VoteStorage{conn: conn}
}

func (s *VoteStorage) CreateVote(vote *models.Vote) (uint32, []string, error) {
	existingVote, errVote := s.GetVote(vote.Id)
	if errVote == nil && existingVote != nil {
		return 0, nil, fmt.Errorf("vote with ID %s already exists", vote.Id)
	}
	if errVote.Error() != "vote not found" {
		return 0, nil, errVote
	}

	expiresAt := uint64(0)
	if vote.ExpiresAt != nil {
		expiresAt = uint64(vote.ExpiresAt.Unix())
	}
	_, err := s.conn.Do(
		tarantool.NewReplaceRequest("votes").
			Tuple([]interface{}{
				vote.Id,
				vote.CreatorId,
				vote.Question,
				vote.Answers,
				vote.Votes,
				vote.IsActive,
				vote.IsMultipleAnswers,
				vote.IsAnonymous,
				expiresAt,
			}),
	).Get()
	if err != nil {
		return 0, []string{}, err
	}
	return vote.Id, vote.Answers, nil
}

func (s *VoteStorage) Vote(voteId uint32, votedId string, answers []string) error {
	vote, err := s.GetVote(voteId)
	if err != nil {
		return err
	}

	updatingVote := vote[0].([]interface{})
	votes, err := mapVotesInterfacesToString(updatingVote[4].(map[interface{}]interface{}))
	if err != nil {
		return err
	}

	expiresAt := time.Unix(int64(updatingVote[8].(uint64)), 0)
	zeroTime := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	if expiresAt.UTC() != zeroTime && time.Now().After(expiresAt) {
		updatingVote[5] = false
		voteModel := models.NewVoteTarantoolModel(
			updatingVote[0],
			updatingVote[1],
			updatingVote[2],
			updatingVote[3],
			updatingVote[4],
			updatingVote[5],
			updatingVote[6],
			updatingVote[7],
			updatingVote[8],
		)
		_, _ = s.conn.Do(
			tarantool.NewReplaceRequest("votes").
				Tuple([]interface{}{
					voteModel.Id,
					voteModel.CreatorId,
					voteModel.Question,
					voteModel.Answers,
					voteModel.Votes,
					voteModel.IsActive,
					voteModel.IsMultipleAnswers,
					voteModel.IsAnonymous,
					voteModel.ExpiresAt,
				}),
		).Get()
		return errors.New("the voting completed")
	}

	if !updatingVote[5].(bool) {
		return errors.New("the voting is already closed")
	}

	allowedAnswers := updatingVote[3].([]interface{})
	var stringAnswers []string
	for _, answer := range allowedAnswers {
		value, ok := answer.(string)
		if !ok {
			return errors.New("invalid map value type")
		}
		stringAnswers = append(stringAnswers, value)
	}

	for _, answer := range answers {
		if !contains(stringAnswers, answer) {
			return errors.New("you can only choose available options")
		}
	}

	if updatingVote[6].(bool) {
		existingVotes := votes[votedId]
		for _, answer := range answers {
			if !contains(existingVotes, answer) {
				existingVotes = append(existingVotes, answer)
			}
		}
		votes[votedId] = existingVotes
	} else {
		if len(answers) > 1 {
			return errors.New("you can only choose one option")
		}
		votes[votedId] = answers
	}

	updatingVote[4] = votes

	voteModel := models.NewVoteTarantoolModel(
		updatingVote[0],
		updatingVote[1],
		updatingVote[2],
		updatingVote[3],
		updatingVote[4],
		updatingVote[5],
		updatingVote[6],
		updatingVote[7],
		updatingVote[8],
	)
	_, err = s.conn.Do(
		tarantool.NewReplaceRequest("votes").
			Tuple([]interface{}{
				voteModel.Id,
				voteModel.CreatorId,
				voteModel.Question,
				voteModel.Answers,
				voteModel.Votes,
				voteModel.IsActive,
				voteModel.IsMultipleAnswers,
				voteModel.IsAnonymous,
				voteModel.ExpiresAt,
			}),
	).Get()

	return err
}

func (s *VoteStorage) GetResults(voteId uint32) (map[string]int, error) {
	vote, err := s.GetVote(voteId)
	if err != nil {
		return nil, err
	}

	votes, err := mapVotesInterfacesToString(vote[0].([]interface{})[4].(map[interface{}]interface{}))
	if err != nil {
		return nil, err
	}

	isAnonymous := vote[0].([]interface{})[7].(bool)
	results := make(map[string]int)

	if isAnonymous {
		for _, options := range votes {
			for _, option := range options {
				results[option]++
			}
		}
	} else {
		for voted, options := range votes {
			for _, option := range options {
				results[fmt.Sprintf("%s: %s", voted, option)]++
			}
		}
	}
	return results, nil
}

func (s *VoteStorage) CloseVote(voteId uint32, creatorId string) error {
	vote, err := s.GetVote(voteId)
	if err != nil {
		return err
	}

	if vote[0].([]interface{})[1] != creatorId {
		return errors.New("only creator can close poll")
	}

	updatedVote := vote[0].([]interface{})
	updatedVote[5] = false

	voteModel := models.NewVoteTarantoolModel(
		updatedVote[0],
		updatedVote[1],
		updatedVote[2],
		updatedVote[3],
		updatedVote[4],
		updatedVote[5],
		updatedVote[6],
		updatedVote[7],
		updatedVote[8],
	)
	_, err = s.conn.Do(tarantool.NewReplaceRequest("votes").
		Tuple([]interface{}{
			voteModel.Id,
			voteModel.CreatorId,
			voteModel.Question,
			voteModel.Answers,
			voteModel.Votes,
			voteModel.IsActive,
			voteModel.IsMultipleAnswers,
			voteModel.IsAnonymous,
			voteModel.ExpiresAt,
		}),
	).Get()
	return err
}

func (s *VoteStorage) DeleteVote(voteId uint32, creatorId string) error {
	vote, err := s.GetVote(voteId)
	if err != nil {
		return err
	}

	if vote[0].([]interface{})[1] != creatorId {
		return errors.New("only creator can delete poll")
	}

	_, err = s.conn.Do(
		tarantool.NewDeleteRequest("votes").
			Index("primary").
			Key([]interface{}{voteId}),
	).Get()
	return err
}

func (s *VoteStorage) GetVote(voteId uint32) ([]interface{}, error) {
	vote, err := s.conn.Do(
		tarantool.NewSelectRequest("votes").
			Index("primary").
			Offset(0).
			Limit(1).
			Iterator(tarantool.IterEq).
			Key([]interface{}{voteId}),
	).Get()

	if err != nil {
		return nil, err
	}

	if len(vote) == 0 {
		return nil, errors.New("vote not found")
	}

	return vote, nil
}

func (s *VoteStorage) AutoCloseExpiredVotes() error {
	votes, err := s.conn.Do(
		tarantool.NewSelectRequest("votes").Iterator(tarantool.IterAll),
	).Get()

	if err != nil {
		return err
	}

	for _, v := range votes {
		vote := v.([]interface{})
		isActive := vote[5].(bool)
		expiresAt := time.Unix(int64(vote[8].(uint64)), 0)
		zeroTime := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
		if expiresAt.UTC() != zeroTime && isActive && time.Now().After(expiresAt) {
			vote[5] = false
			voteModel := models.NewVoteTarantoolModel(
				vote[0],
				vote[1],
				vote[2],
				vote[3],
				vote[4],
				vote[5],
				vote[6],
				vote[7],
				vote[8],
			)
			_, _ = s.conn.Do(
				tarantool.NewReplaceRequest("votes").
					Tuple([]interface{}{
						voteModel.Id,
						voteModel.CreatorId,
						voteModel.Question,
						voteModel.Answers,
						voteModel.Votes,
						voteModel.IsActive,
						voteModel.IsMultipleAnswers,
						voteModel.IsAnonymous,
						voteModel.ExpiresAt,
					}),
			).Get()
		}
	}

	return nil
}

func (s *VoteStorage) Close() error {
	return s.conn.Close()
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func mapVotesInterfacesToString(votes map[interface{}]interface{}) (map[string][]string, error) {
	stringMap := make(map[string][]string)
	for k, v := range votes {
		key, ok := k.(string)
		if !ok {
			return nil, fmt.Errorf("invalid map key type: %T", k)
		}
		value, ok := v.([]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid map value type: %T", v)
		}

		var strValues []string
		for _, item := range value {
			strValue, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("invalid value type in array: %T", item)
			}
			strValues = append(strValues, strValue)
		}
		stringMap[key] = strValues
	}

	return stringMap, nil
}
