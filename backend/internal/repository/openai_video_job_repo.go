package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type openAIVideoJobRepository struct {
	db *sql.DB
}

func NewOpenAIVideoJobRepository(db *sql.DB) service.OpenAIVideoJobRepository {
	return &openAIVideoJobRepository{db: db}
}

func (r *openAIVideoJobRepository) GetByVideoID(ctx context.Context, videoID string) (*service.OpenAIVideoJob, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT video_id, platform, account_id, group_id, api_key_id, user_id, channel_id,
       request_model, upstream_model, status, response_json, created_at, updated_at
FROM openai_video_jobs
WHERE video_id = $1`, videoID)
	job, err := scanOpenAIVideoJob(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrOpenAIVideoJobNotFound
		}
		return nil, err
	}
	return job, nil
}

func (r *openAIVideoJobRepository) List(ctx context.Context, filter service.OpenAIVideoJobListFilter) ([]service.OpenAIVideoJob, error) {
	clauses := []string{"1 = 1"}
	args := []any{}
	addArg := func(value any) string {
		args = append(args, value)
		return "$" + strconv.Itoa(len(args))
	}

	if platform := strings.TrimSpace(filter.Platform); platform != "" {
		clauses = append(clauses, "platform = "+addArg(platform))
	}
	if filter.GroupID != nil {
		clauses = append(clauses, "group_id = "+addArg(*filter.GroupID))
	}
	if filter.UserID != nil {
		clauses = append(clauses, "user_id = "+addArg(*filter.UserID))
	}
	order := "DESC"
	if strings.EqualFold(filter.Order, "asc") {
		order = "ASC"
	}
	if after := strings.TrimSpace(filter.After); after != "" {
		afterPlaceholder := addArg(after)
		operator := "<"
		if order == "ASC" {
			operator = ">"
		}
		clauses = append(clauses, `(created_at, video_id) `+operator+` (
            SELECT created_at, video_id FROM openai_video_jobs WHERE video_id = `+afterPlaceholder+`
        )`)
	}
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	args = append(args, limit)
	limitPlaceholder := "$" + strconv.Itoa(len(args))

	query := `
SELECT video_id, platform, account_id, group_id, api_key_id, user_id, channel_id,
       request_model, upstream_model, status, response_json, created_at, updated_at
FROM openai_video_jobs
WHERE ` + strings.Join(clauses, " AND ") + `
ORDER BY created_at ` + order + `, video_id ` + order + `
LIMIT ` + limitPlaceholder

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []service.OpenAIVideoJob{}
	for rows.Next() {
		job, err := scanOpenAIVideoJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *openAIVideoJobRepository) Upsert(ctx context.Context, job *service.OpenAIVideoJob) error {
	if job == nil {
		return errors.New("openai video job is nil")
	}
	responseJSON, err := json.Marshal(job.ResponseJSON)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
INSERT INTO openai_video_jobs (
    video_id, platform, account_id, group_id, api_key_id, user_id, channel_id,
    request_model, upstream_model, status, response_json
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb)
ON CONFLICT (video_id) DO UPDATE SET
    platform = EXCLUDED.platform,
    account_id = EXCLUDED.account_id,
    group_id = EXCLUDED.group_id,
    api_key_id = EXCLUDED.api_key_id,
    user_id = EXCLUDED.user_id,
    channel_id = EXCLUDED.channel_id,
    request_model = EXCLUDED.request_model,
    upstream_model = EXCLUDED.upstream_model,
    status = EXCLUDED.status,
    response_json = EXCLUDED.response_json,
    updated_at = NOW()`,
		job.VideoID,
		job.Platform,
		job.AccountID,
		nullInt64(job.GroupID),
		nullInt64(job.APIKeyID),
		nullInt64(job.UserID),
		nullInt64(job.ChannelID),
		job.RequestModel,
		job.UpstreamModel,
		job.Status,
		string(responseJSON),
	)
	return err
}

func (r *openAIVideoJobRepository) DeleteByVideoID(ctx context.Context, videoID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM openai_video_jobs WHERE video_id = $1`, videoID)
	return err
}

func (r *openAIVideoJobRepository) GetCharacterByID(ctx context.Context, characterID string) (*service.OpenAIVideoCharacter, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT character_id, platform, account_id, group_id, api_key_id, user_id,
       response_json, created_at, updated_at
FROM openai_video_characters
WHERE character_id = $1`, characterID)
	character, err := scanOpenAIVideoCharacter(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrOpenAIVideoCharacterNotFound
		}
		return nil, err
	}
	return character, nil
}

func (r *openAIVideoJobRepository) UpsertCharacter(ctx context.Context, character *service.OpenAIVideoCharacter) error {
	if character == nil {
		return errors.New("openai video character is nil")
	}
	responseJSON, err := json.Marshal(character.ResponseJSON)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
INSERT INTO openai_video_characters (
    character_id, platform, account_id, group_id, api_key_id, user_id, response_json
) VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
ON CONFLICT (character_id) DO UPDATE SET
    platform = EXCLUDED.platform,
    account_id = EXCLUDED.account_id,
    group_id = EXCLUDED.group_id,
    api_key_id = EXCLUDED.api_key_id,
    user_id = EXCLUDED.user_id,
    response_json = EXCLUDED.response_json,
    updated_at = NOW()`,
		character.CharacterID,
		character.Platform,
		character.AccountID,
		nullInt64(character.GroupID),
		nullInt64(character.APIKeyID),
		nullInt64(character.UserID),
		string(responseJSON),
	)
	return err
}

type openAIVideoJobScanner interface {
	Scan(dest ...any) error
}

func scanOpenAIVideoJob(scanner openAIVideoJobScanner) (*service.OpenAIVideoJob, error) {
	var job service.OpenAIVideoJob
	var groupID, apiKeyID, userID, channelID sql.NullInt64
	var responseJSONRaw []byte
	if err := scanner.Scan(
		&job.VideoID,
		&job.Platform,
		&job.AccountID,
		&groupID,
		&apiKeyID,
		&userID,
		&channelID,
		&job.RequestModel,
		&job.UpstreamModel,
		&job.Status,
		&responseJSONRaw,
		&job.CreatedAt,
		&job.UpdatedAt,
	); err != nil {
		return nil, err
	}
	job.GroupID = nullableInt64Ptr(groupID)
	job.APIKeyID = nullableInt64Ptr(apiKeyID)
	job.UserID = nullableInt64Ptr(userID)
	job.ChannelID = nullableInt64Ptr(channelID)
	if len(responseJSONRaw) > 0 {
		_ = json.Unmarshal(responseJSONRaw, &job.ResponseJSON)
	}
	if job.ResponseJSON == nil {
		job.ResponseJSON = map[string]any{}
	}
	return &job, nil
}

func nullableInt64Ptr(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	value := v.Int64
	return &value
}

type openAIVideoCharacterScanner interface {
	Scan(dest ...any) error
}

func scanOpenAIVideoCharacter(scanner openAIVideoCharacterScanner) (*service.OpenAIVideoCharacter, error) {
	var character service.OpenAIVideoCharacter
	var groupID, apiKeyID, userID sql.NullInt64
	var responseJSONRaw []byte
	if err := scanner.Scan(
		&character.CharacterID,
		&character.Platform,
		&character.AccountID,
		&groupID,
		&apiKeyID,
		&userID,
		&responseJSONRaw,
		&character.CreatedAt,
		&character.UpdatedAt,
	); err != nil {
		return nil, err
	}
	character.GroupID = nullableInt64Ptr(groupID)
	character.APIKeyID = nullableInt64Ptr(apiKeyID)
	character.UserID = nullableInt64Ptr(userID)
	if len(responseJSONRaw) > 0 {
		_ = json.Unmarshal(responseJSONRaw, &character.ResponseJSON)
	}
	if character.ResponseJSON == nil {
		character.ResponseJSON = map[string]any{}
	}
	return &character, nil
}
