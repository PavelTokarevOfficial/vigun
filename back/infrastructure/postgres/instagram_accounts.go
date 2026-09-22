package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/finde-clip/finde-v2/back/internal/instagram"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InstagramAccounts struct {
	db *pgxpool.Pool
}

func NewInstagramAccounts(db *pgxpool.Pool) *InstagramAccounts {
	return &InstagramAccounts{db: db}
}

func (r *InstagramAccounts) List(ctx context.Context) ([]instagram.Account, error) {
	rows, err := r.db.Query(ctx, `SELECT id,nickname,instagram_user_id,access_token <> '',token_updated_at,
		token_expires_at,token_last_checked_at,COALESCE(verified_username,''),COALESCE(account_type,'')
		FROM instagram_accounts ORDER BY nickname,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []instagram.Account{}
	for rows.Next() {
		var item instagram.Account
		if err = scanInstagramAccount(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *InstagramAccounts) Create(ctx context.Context, input instagram.AccountInput) (instagram.Account, error) {
	var item instagram.Account
	row := r.db.QueryRow(ctx, `INSERT INTO instagram_accounts(nickname,instagram_user_id,access_token)
		VALUES($1,$2,$3)
		RETURNING id,nickname,instagram_user_id,access_token <> '',token_updated_at,
			token_expires_at,token_last_checked_at,COALESCE(verified_username,''),COALESCE(account_type,'')`,
		input.Nickname, input.InstagramUserID, input.AccessToken,
	)
	err := scanInstagramAccount(row, &item)
	return item, err
}

func (r *InstagramAccounts) Update(ctx context.Context, id string, input instagram.AccountInput) (instagram.Account, error) {
	var item instagram.Account
	row := r.db.QueryRow(ctx, `UPDATE instagram_accounts SET
		nickname=$2,instagram_user_id=$3,
		access_token=CASE WHEN $4='' THEN access_token ELSE $4 END,
		token_updated_at=CASE WHEN $4='' THEN token_updated_at ELSE now() END,
		token_expires_at=CASE WHEN $4='' THEN token_expires_at ELSE NULL END,
		token_last_checked_at=CASE WHEN $4='' THEN token_last_checked_at ELSE NULL END,
		updated_at=now()
		WHERE id=$1
		RETURNING id,nickname,instagram_user_id,access_token <> '',token_updated_at,
			token_expires_at,token_last_checked_at,COALESCE(verified_username,''),COALESCE(account_type,'')`,
		id, input.Nickname, input.InstagramUserID, input.AccessToken,
	)
	err := scanInstagramAccount(row, &item)
	if err == pgx.ErrNoRows {
		return instagram.Account{}, fmt.Errorf("Instagram account not found")
	}
	return item, err
}

func (r *InstagramAccounts) Delete(ctx context.Context, id string) error {
	result, err := r.db.Exec(ctx, `DELETE FROM instagram_accounts WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("Instagram account not found")
	}
	return nil
}

func (r *InstagramAccounts) Credentials(ctx context.Context, id string) (instagram.Credentials, error) {
	var credentials instagram.Credentials
	err := r.db.QueryRow(ctx, `SELECT instagram_user_id,access_token
		FROM instagram_accounts WHERE id=$1`, id,
	).Scan(&credentials.InstagramUserID, &credentials.AccessToken)
	if err == pgx.ErrNoRows {
		return instagram.Credentials{}, fmt.Errorf("Instagram account not found")
	}
	return credentials, err
}

func (r *InstagramAccounts) UpdateToken(ctx context.Context, id, accessToken string, expiresAt time.Time) (instagram.Account, error) {
	var item instagram.Account
	row := r.db.QueryRow(ctx, `UPDATE instagram_accounts SET
		access_token=$2,token_updated_at=now(),token_expires_at=$3,
		token_last_checked_at=NULL,updated_at=now()
		WHERE id=$1
		RETURNING id,nickname,instagram_user_id,access_token <> '',token_updated_at,
			token_expires_at,token_last_checked_at,COALESCE(verified_username,''),COALESCE(account_type,'')`,
		id, accessToken, expiresAt,
	)
	err := scanInstagramAccount(row, &item)
	if err == pgx.ErrNoRows {
		return instagram.Account{}, fmt.Errorf("Instagram account not found")
	}
	return item, err
}

func (r *InstagramAccounts) MarkTokenChecked(ctx context.Context, id string, profile instagram.TokenProfile) (instagram.Account, error) {
	var item instagram.Account
	row := r.db.QueryRow(ctx, `UPDATE instagram_accounts SET
		token_last_checked_at=now(),verified_username=$2,account_type=$3,updated_at=now()
		WHERE id=$1
		RETURNING id,nickname,instagram_user_id,access_token <> '',token_updated_at,
			token_expires_at,token_last_checked_at,COALESCE(verified_username,''),COALESCE(account_type,'')`,
		id, profile.Username, profile.AccountType,
	)
	err := scanInstagramAccount(row, &item)
	if err == pgx.ErrNoRows {
		return instagram.Account{}, fmt.Errorf("Instagram account not found")
	}
	return item, err
}

type accountScanner interface {
	Scan(...any) error
}

func scanInstagramAccount(row accountScanner, item *instagram.Account) error {
	return row.Scan(
		&item.ID,
		&item.Nickname,
		&item.InstagramUserID,
		&item.HasAccessToken,
		&item.TokenUpdatedAt,
		&item.TokenExpiresAt,
		&item.TokenLastCheckedAt,
		&item.VerifiedUsername,
		&item.AccountType,
	)
}
