package main

import (
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type CreateAccountRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Password  string `json:"password"`
}

type LoginRequest struct {
	Number   int64  `json:"number"`
	Password string `json:"password"`
}
type LoginReponse struct {
	Number int64  `json:"number"`
	Token  string `json:"token"`
}

type TransferRequest struct {
	ToAccount int `json:"toAccount"`
	Amount    int `json:"amount"`
}

type Account struct {
	ID                int       `json:"id"`
	FirstName         string    `json:"firstName"`
	LastName          string    `json:"lastName"`
	Number            int64     `json:"number"`
	Balance           int64     `json:"balance"`
	CreatedAt         time.Time `json:"createdat"`
	EncryptedPassword string    `json:"-"`
}

func (a *Account) ValidatePW(pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(a.EncryptedPassword), []byte(pw)) == nil

}

func NewAccount(firstname, lastname, password string) (*Account, error) {
	encpw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &Account{
		FirstName:         firstname,
		LastName:          lastname,
		Number:            int64(rand.Intn(100000)),
		EncryptedPassword: string(encpw),
		CreatedAt:         time.Now().UTC(),
	}, nil
}
