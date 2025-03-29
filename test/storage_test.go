package test

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/tarantool/go-tarantool/v2"
	"mattermost-voting-bot/internal/models"
	"mattermost-voting-bot/internal/storage"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestCreateVote(t *testing.T) {
	setup := setupDockerContainer(t)
	defer setup()

	conn, err := newConnection()
	assert.NoError(t, err)
	defer conn.Close()

	voteStorage := storage.NewVoteStorage(conn)

	vote := &models.Vote{
		Id:                123,
		CreatorId:         "user1",
		Question:          "Your favorite language?",
		Answers:           []string{"Go", "Python"},
		Votes:             make(map[string][]string),
		IsActive:          true,
		IsMultipleAnswers: false,
		IsAnonymous:       false,
		ExpiresAt:         nil,
	}

	id, answers, err := voteStorage.CreateVote(vote)
	assert.NoError(t, err)
	assert.Equal(t, uint32(123), id)
	assert.Equal(t, []string{"Go", "Python"}, answers)
}

func TestCreateExistingVote(t *testing.T) {
	setup := setupDockerContainer(t)
	defer setup()

	conn, err := newConnection()
	assert.NoError(t, err)
	defer conn.Close()

	voteStorage := storage.NewVoteStorage(conn)

	vote := &models.Vote{
		Id:                123,
		CreatorId:         "user1",
		Question:          "Your favorite language?",
		Answers:           []string{"Go", "Python"},
		Votes:             make(map[string][]string),
		IsActive:          true,
		IsMultipleAnswers: false,
		IsAnonymous:       false,
		ExpiresAt:         nil,
	}

	_, _, err = voteStorage.CreateVote(vote)

	vote = &models.Vote{
		Id:                123,
		CreatorId:         "user2",
		Question:          "?egaugnal etirovaf ruoY",
		Answers:           []string{"oG", "nohtyP"},
		Votes:             make(map[string][]string),
		IsActive:          true,
		IsMultipleAnswers: false,
		IsAnonymous:       false,
		ExpiresAt:         nil,
	}

	_, _, err = voteStorage.CreateVote(vote)
	assert.EqualError(t, err, "vote with ID %!s(uint32=123) already exists")
}

func TestCreateVoteWithExpiresAtField(t *testing.T) {
	setup := setupDockerContainer(t)
	defer setup()

	conn, err := newConnection()
	assert.NoError(t, err)
	defer conn.Close()

	voteStorage := storage.NewVoteStorage(conn)

	timeVar := time.Now().Add(time.Hour).Truncate(time.Second)
	expiresAt := &timeVar
	var voteId uint32 = 123
	vote := &models.Vote{
		Id:                voteId,
		CreatorId:         "user1",
		Question:          "Your favorite language?",
		Answers:           []string{"Go", "Python"},
		Votes:             make(map[string][]string),
		IsActive:          true,
		IsMultipleAnswers: false,
		IsAnonymous:       false,
		ExpiresAt:         expiresAt,
	}

	_, _, err = voteStorage.CreateVote(vote)
	assert.NoError(t, err)

	getVote, err := voteStorage.GetVote(voteId)
	assert.NoError(t, err)

	voteExpiresAt := time.Unix(int64(getVote[0].([]interface{})[8].(uint64)), 0)
	assert.NoError(t, err)
	assert.Equal(t, expiresAt, &voteExpiresAt)
}

func TestVote(t *testing.T) {
	setup := setupDockerContainer(t)
	defer setup()

	conn, err := newConnection()
	assert.NoError(t, err)
	defer conn.Close()

	voteStorage := storage.NewVoteStorage(conn)

	var voteId uint32 = 123
	vote := &models.Vote{
		Id:                voteId,
		CreatorId:         "user1",
		Question:          "Your favorite language?",
		Answers:           []string{"Go", "Python"},
		Votes:             make(map[string][]string),
		IsActive:          true,
		IsMultipleAnswers: false,
		IsAnonymous:       false,
		ExpiresAt:         nil,
	}

	_, _, err = voteStorage.CreateVote(vote)
	assert.NoError(t, err)

	err = voteStorage.Vote(voteId, "user2", []string{"Go"})
	assert.NoError(t, err)

	getVote, err := voteStorage.GetVote(voteId)
	assert.NoError(t, err)

	updatedVote := getVote[0].([]interface{})
	votesInterface := updatedVote[4].(map[interface{}]interface{})
	votes := make(map[string][]string)
	for key, value := range votesInterface {
		strKey, ok := key.(string)
		if !ok {
			assert.NoError(t, err)
		}
		strValue, ok := value.([]interface{})
		if !ok {
			t.Errorf("Expected value to be []interface{}, got %T", value)
			continue
		}
		var strValues []string
		for _, v := range strValue {
			s, ok := v.(string)
			if !ok {
				t.Errorf("Expected value in slice to be string, got %T", v)
				continue
			}
			strValues = append(strValues, s)
		}

		votes[strKey] = strValues
	}
	assert.NoError(t, err)
	assert.Equal(t, map[string][]string{"user2": {"Go"}}, votes)
}

func TestVoteCompleted(t *testing.T) {
	setup := setupDockerContainer(t)
	defer setup()

	conn, err := newConnection()
	assert.NoError(t, err)
	defer conn.Close()

	voteStorage := storage.NewVoteStorage(conn)

	timeVar := time.Now().Add(-1 * time.Hour).Truncate(time.Second)
	expiresAt := &timeVar
	var voteId uint32 = 123
	vote := &models.Vote{
		Id:                voteId,
		CreatorId:         "user1",
		Question:          "Your favorite language?",
		Answers:           []string{"Go", "Python"},
		Votes:             make(map[string][]string),
		IsActive:          true,
		IsMultipleAnswers: false,
		IsAnonymous:       false,
		ExpiresAt:         expiresAt,
	}

	_, _, err = voteStorage.CreateVote(vote)
	assert.NoError(t, err)

	err = voteStorage.Vote(voteId, "user2", []string{"Go"})
	assert.EqualError(t, err, "the voting completed")
}

func TestVoteClosed(t *testing.T) {
	setup := setupDockerContainer(t)
	defer setup()

	conn, err := newConnection()
	assert.NoError(t, err)
	defer conn.Close()

	voteStorage := storage.NewVoteStorage(conn)

	var voteId uint32 = 123
	vote := &models.Vote{
		Id:                voteId,
		CreatorId:         "user1",
		Question:          "Your favorite language?",
		Answers:           []string{"Go", "Python"},
		Votes:             make(map[string][]string),
		IsActive:          false,
		IsMultipleAnswers: false,
		IsAnonymous:       false,
		ExpiresAt:         nil,
	}

	_, _, err = voteStorage.CreateVote(vote)
	assert.NoError(t, err)

	err = voteStorage.Vote(voteId, "user2", []string{"Go"})
	assert.EqualError(t, err, "the voting is already closed")
}

func TestVoteWithUnavailableAnswer(t *testing.T) {
	setup := setupDockerContainer(t)
	defer setup()

	conn, err := newConnection()
	assert.NoError(t, err)
	defer conn.Close()

	voteStorage := storage.NewVoteStorage(conn)

	var voteId uint32 = 123
	vote := &models.Vote{
		Id:                voteId,
		CreatorId:         "user1",
		Question:          "Your favorite language?",
		Answers:           []string{"Go", "Python"},
		Votes:             make(map[string][]string),
		IsActive:          true,
		IsMultipleAnswers: false,
		IsAnonymous:       false,
		ExpiresAt:         nil,
	}

	_, _, err = voteStorage.CreateVote(vote)
	assert.NoError(t, err)

	err = voteStorage.Vote(voteId, "user2", []string{"Java"})
	assert.EqualError(t, err, "you can only choose available options")
}

func TestVoteWithOneAnswer(t *testing.T) {
	setup := setupDockerContainer(t)
	defer setup()

	conn, err := newConnection()
	assert.NoError(t, err)
	defer conn.Close()

	voteStorage := storage.NewVoteStorage(conn)

	var voteId uint32 = 123
	vote := &models.Vote{
		Id:                voteId,
		CreatorId:         "user1",
		Question:          "Your favorite language?",
		Answers:           []string{"Go", "Python"},
		Votes:             make(map[string][]string),
		IsActive:          true,
		IsMultipleAnswers: false,
		IsAnonymous:       false,
		ExpiresAt:         nil,
	}

	_, _, err = voteStorage.CreateVote(vote)
	assert.NoError(t, err)

	err = voteStorage.Vote(voteId, "user2", []string{"Go", "Python"})
	assert.EqualError(t, err, "you can only choose one option")
}

func TestGetResults(t *testing.T) {
	setup := setupDockerContainer(t)
	defer setup()

	conn, err := newConnection()
	assert.NoError(t, err)
	defer conn.Close()

	voteStorage := storage.NewVoteStorage(conn)

	var voteId uint32 = 123
	vote := &models.Vote{
		Id:                voteId,
		CreatorId:         "user1",
		Question:          "Your favorite language?",
		Answers:           []string{"Go", "Python"},
		Votes:             make(map[string][]string),
		IsActive:          true,
		IsMultipleAnswers: false,
		IsAnonymous:       false,
		ExpiresAt:         nil,
	}

	_, _, err = voteStorage.CreateVote(vote)
	assert.NoError(t, err)

	err = voteStorage.Vote(voteId, "user2", []string{"Go"})
	assert.NoError(t, err)

	results, err := voteStorage.GetResults(voteId)
	assert.NoError(t, err)

	assert.Equal(t, map[string]int{"user2: Go": 1}, results)
}

func TestCloseVote(t *testing.T) {
	setup := setupDockerContainer(t)
	defer setup()

	conn, err := newConnection()
	assert.NoError(t, err)
	defer conn.Close()

	voteStorage := storage.NewVoteStorage(conn)

	var voteId uint32 = 123
	vote := &models.Vote{
		Id:                voteId,
		CreatorId:         "user1",
		Question:          "Your favorite language?",
		Answers:           []string{"Go", "Python"},
		Votes:             make(map[string][]string),
		IsActive:          true,
		IsMultipleAnswers: false,
		IsAnonymous:       false,
		ExpiresAt:         nil,
	}

	_, _, err = voteStorage.CreateVote(vote)
	assert.NoError(t, err)

	err = voteStorage.CloseVote(voteId, "user1")
	assert.NoError(t, err)

	getVote, err := voteStorage.GetVote(voteId)
	assert.NoError(t, err)
	isActive := getVote[0].([]interface{})[5].(bool)
	assert.NoError(t, err)
	assert.Equal(t, false, isActive)
}

func TestCloseVoteByNotCreator(t *testing.T) {
	setup := setupDockerContainer(t)
	defer setup()

	conn, err := newConnection()
	assert.NoError(t, err)
	defer conn.Close()

	voteStorage := storage.NewVoteStorage(conn)

	var voteId uint32 = 123
	vote := &models.Vote{
		Id:                voteId,
		CreatorId:         "user1",
		Question:          "Your favorite language?",
		Answers:           []string{"Go", "Python"},
		Votes:             make(map[string][]string),
		IsActive:          true,
		IsMultipleAnswers: false,
		IsAnonymous:       false,
		ExpiresAt:         nil,
	}

	_, _, err = voteStorage.CreateVote(vote)
	assert.NoError(t, err)

	err = voteStorage.CloseVote(voteId, "user2")
	assert.EqualError(t, err, "only creator can close poll")
}

func TestDeleteVote(t *testing.T) {
	setup := setupDockerContainer(t)
	defer setup()

	conn, err := newConnection()
	assert.NoError(t, err)
	defer conn.Close()

	voteStorage := storage.NewVoteStorage(conn)

	var voteId uint32 = 123
	vote := &models.Vote{
		Id:                voteId,
		CreatorId:         "user1",
		Question:          "Your favorite language?",
		Answers:           []string{"Go", "Python"},
		Votes:             make(map[string][]string),
		IsActive:          true,
		IsMultipleAnswers: false,
		IsAnonymous:       false,
		ExpiresAt:         nil,
	}

	_, _, err = voteStorage.CreateVote(vote)
	assert.NoError(t, err)

	err = voteStorage.DeleteVote(voteId, "user1")
	assert.NoError(t, err)

	getVote, err := voteStorage.GetVote(voteId)
	assert.Error(t, err)
	assert.Nil(t, getVote)
	assert.EqualError(t, err, "vote not found")
}

func TestDeleteVoteByNotCreator(t *testing.T) {
	setup := setupDockerContainer(t)
	defer setup()

	conn, err := newConnection()
	assert.NoError(t, err)
	defer conn.Close()

	voteStorage := storage.NewVoteStorage(conn)

	var voteId uint32 = 123
	vote := &models.Vote{
		Id:                voteId,
		CreatorId:         "user1",
		Question:          "Your favorite language?",
		Answers:           []string{"Go", "Python"},
		Votes:             make(map[string][]string),
		IsActive:          true,
		IsMultipleAnswers: false,
		IsAnonymous:       false,
		ExpiresAt:         nil,
	}

	_, _, err = voteStorage.CreateVote(vote)
	assert.NoError(t, err)

	err = voteStorage.DeleteVote(voteId, "user2")
	assert.EqualError(t, err, "only creator can delete poll")
}

func TestAutoCloseExpiredVotes(t *testing.T) {
	setup := setupDockerContainer(t)
	defer setup()

	conn, err := newConnection()
	assert.NoError(t, err)
	defer conn.Close()

	voteStorage := storage.NewVoteStorage(conn)

	timeVar := time.Now().Add(-1 * time.Hour).Truncate(time.Second)
	expiresAt := &timeVar
	var voteId uint32 = 123
	vote := &models.Vote{
		Id:                voteId,
		CreatorId:         "user1",
		Question:          "Your favorite language?",
		Answers:           []string{"Go", "Python"},
		Votes:             make(map[string][]string),
		IsActive:          true,
		IsMultipleAnswers: false,
		IsAnonymous:       false,
		ExpiresAt:         expiresAt,
	}

	_, _, err = voteStorage.CreateVote(vote)
	assert.NoError(t, err)

	err = voteStorage.AutoCloseExpiredVotes()
	assert.NoError(t, err)
	getVote, err := voteStorage.GetVote(voteId)
	assert.NoError(t, err)
	isActive := getVote[0].([]interface{})[5].(bool)
	assert.NoError(t, err)
	assert.Equal(t, false, isActive)
}

func newConnection() (*tarantool.Connection, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	dialer := tarantool.NetDialer{
		Address: "127.0.0.1:3302",
		User:    "guest",
	}
	opts := tarantool.Opts{
		Timeout: 5 * time.Second,
	}

	conn, err := tarantool.Connect(ctx, dialer, opts)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func startDockerCompose() error {
	cmd := exec.Command("docker-compose", "-f", "./docker_test/docker-compose.yml", "up", "-d")
	cmd.Env = append(os.Environ(), "FUNCNEST=1000")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to start Docker Compose: %v", err)
	}
	containerName := " test_tarantool"
	for {
		psCmd := exec.Command("docker", "ps", "-f", fmt.Sprintf("name=%s", containerName))
		psCmd.Stdout = os.Stdout
		psCmd.Stderr = os.Stderr
		err := psCmd.Run()

		if err == nil {
			break
		}

		time.Sleep(2 * time.Second)
	}
	return nil
}

func stopDockerCompose() error {
	cmd := exec.Command("docker-compose", "-f", "./docker_test/docker-compose.yml", "down")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to stop Docker Compose: %v", err)
	}
	return nil
}

func deleteDockerVolume() error {
	cmd := exec.Command("sh", "-c", "docker volume rm docker_test_tarantool_data 2>/dev/null || true")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to delete docker volume: %v", err)
	}
	return nil
}

func setupDockerContainer(t *testing.T) func() {
	err := deleteDockerVolume()
	if err != nil {
		t.Fatalf("Failed to delete docker volume: %v", err)
	}

	err = startDockerCompose()
	if err != nil {
		t.Fatalf("Failed to start Docker Compose: %v", err)
	}

	return func() {
		err := stopDockerCompose()
		if err != nil {
			t.Errorf("Failed to stop Docker Compose: %v", err)
		}
	}
}
