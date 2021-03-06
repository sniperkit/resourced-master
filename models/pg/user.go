package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"

	"github.com/resourced/resourced-master/libstring"
)

func NewUser(ctx context.Context) *User {
	user := &User{}
	user.AppContext = ctx
	user.table = "users"
	user.hasID = true
	user.i = user

	return user
}

type UserRow struct {
	ID                     int64          `db:"id"`
	Email                  string         `db:"email"`
	Password               string         `db:"password"`
	EmailVerificationToken sql.NullString `db:"email_verification_token"`
	EmailVerified          bool           `db:"email_verified"`
}

type User struct {
	Base
}

func (u *User) userRowFromSqlResult(tx *sqlx.Tx, sqlResult sql.Result) (*UserRow, error) {
	userId, err := sqlResult.LastInsertId()
	if err != nil {
		return nil, err
	}

	return u.GetByID(tx, userId)
}

// All returns all user rows.
func (u *User) All(tx *sqlx.Tx) ([]*UserRow, error) {
	pgdb, err := u.GetPGDB()
	if err != nil {
		return nil, err
	}

	users := []*UserRow{}
	query := fmt.Sprintf("SELECT * FROM %v", u.table)
	err = pgdb.Select(&users, query)

	return users, err
}

// GetByID returns record by id.
func (u *User) GetByID(tx *sqlx.Tx, id int64) (*UserRow, error) {
	pgdb, err := u.GetPGDB()
	if err != nil {
		return nil, err
	}

	user := &UserRow{}
	query := fmt.Sprintf("SELECT * FROM %v WHERE id=$1", u.table)
	err = pgdb.Get(user, query, id)

	return user, err
}

// GetByEmail returns record by email.
func (u *User) GetByEmail(tx *sqlx.Tx, email string) (*UserRow, error) {
	pgdb, err := u.GetPGDB()
	if err != nil {
		return nil, err
	}

	user := &UserRow{}
	query := fmt.Sprintf("SELECT * FROM %v WHERE email=$1", u.table)
	err = pgdb.Get(user, query, email)

	return user, err
}

// GetByEmailVerificationToken returns record by email_verification_token.
func (u *User) GetByEmailVerificationToken(tx *sqlx.Tx, emailVerificationToken string) (*UserRow, error) {
	pgdb, err := u.GetPGDB()
	if err != nil {
		return nil, err
	}

	user := &UserRow{}
	query := fmt.Sprintf("SELECT * FROM %v WHERE email_verification_token=$1", u.table)
	err = pgdb.Get(user, query, emailVerificationToken)

	return user, err
}

// GetByEmail returns record by email but checks password first.
func (u *User) GetUserByEmailAndPassword(tx *sqlx.Tx, email, password string) (*UserRow, error) {
	user, err := u.GetByEmail(tx, email)
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, err
	}

	return user, err
}

// SignupRandomPassword create a new record of user with random password.
func (u *User) SignupRandomPassword(tx *sqlx.Tx, email string) (*UserRow, error) {
	password, _ := libstring.GeneratePassword(32)
	passwordAgain := password

	return u.Signup(tx, email, password, passwordAgain)
}

// Signup create a new record of user.
func (u *User) Signup(tx *sqlx.Tx, email, password, passwordAgain string) (*UserRow, error) {
	if email == "" {
		return nil, errors.New("Email cannot be blank.")
	}
	if password == "" {
		return nil, errors.New("Password cannot be blank.")
	}
	if password != passwordAgain {
		return nil, errors.New("Password is invalid.")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 5)
	if err != nil {
		return nil, err
	}

	emailVerificationToken, err := libstring.GeneratePassword(32)
	if err != nil {
		return nil, err
	}

	data := make(map[string]interface{})
	data["email"] = email
	data["password"] = hashedPassword
	data["email_verification_token"] = emailVerificationToken

	sqlResult, err := u.InsertIntoTable(tx, data)
	if err != nil {
		return nil, err
	}

	return u.userRowFromSqlResult(tx, sqlResult)
}

// UpdateEmailAndPasswordByID updates user email and password.
func (u *User) UpdateEmailAndPasswordByID(tx *sqlx.Tx, userId int64, email, password, passwordAgain string) (*UserRow, error) {
	data := make(map[string]interface{})

	if email != "" {
		data["email"] = email
	}

	if password != "" && passwordAgain != "" && password == passwordAgain {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 5)
		if err != nil {
			return nil, err
		}

		data["password"] = hashedPassword
	}

	if len(data) > 0 {
		_, err := u.UpdateByID(tx, data, userId)
		if err != nil {
			return nil, err
		}
	}

	return u.GetByID(tx, userId)
}

// UpdateEmailVerification acknowledge email verification.
func (u *User) UpdateEmailVerification(tx *sqlx.Tx, emailVerificationToken string) (*UserRow, error) {
	if emailVerificationToken == "" {
		return nil, errors.New("Token cannot be empty")
	}

	existingUser, err := u.GetByEmailVerificationToken(tx, emailVerificationToken)
	if err != nil {
		return nil, err
	}

	data := make(map[string]interface{})
	data["email_verification_token"] = ""
	data["email_verified"] = true

	if len(data) > 0 {
		_, err := u.UpdateByKeyValueString(tx, data, "email_verification_token", emailVerificationToken)
		if err != nil {
			return nil, err
		}
	}

	return u.GetByID(tx, existingUser.ID)
}
