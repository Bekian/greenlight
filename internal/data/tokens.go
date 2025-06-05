package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"time"

	"github.com/Bekian/greenlight/internal/validator"
)

// constants for token scope
const (
	ScopeActivation = "activation"
)

// data relating to an individual token
type Token struct {
	Plaintext string
	Hash      []byte
	UserId    int64
	Expiry    time.Time
	Scope     string
}

// DIFF Note: slightly different casing for userID
func generateToken(userId int64, ttl time.Duration, scope string) *Token {
	token := &Token{
		Plaintext: rand.Text(),
		UserId:    userId,
		Expiry:    time.Now().Add(ttl),
		Scope:     scope,
	}

	// generate hash, the [:] syntax converts the array to a slice
	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token
}

// validate token is 26 bytes long
// BEK Note: is there an instance where this wouldn't be the case?
func ValidateTokenPlaintext(v *validator.Validator, tokenPlaintext string) {
	// remember the condition must be TRUE for the check to be "ok"
	v.Check(tokenPlaintext != "", "token", "must be provided")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 26 bytes long")
}

type TokenModel struct {
	DB *sql.DB
}

// generate and insert token into tokens table
func (m TokenModel) New(userId int64, ttl time.Duration, scope string) (*Token, error) {
	token := generateToken(userId, ttl, scope)

	err := m.Insert(token)
	return token, err
}

// insert token record into tokens table
func (m TokenModel) Insert(token *Token) error {
	query := `
		INSERT INTO tokens (hash, user_id, expiry, scope)
		VALUES ($1, $2, $3, $4)`

	args := []any{token.Hash, token.UserId, token.Expiry, token.Scope}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, args...)
	return err
}

// delete all tokens for a specific user and scope
func (m TokenModel) DeleteAllForUser(scope string, userId int64) error {
	query := `
		DELETE FROM tokens
		WHERE scope = $1 AND user_id = $2`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, scope, userId)
	return err

}
