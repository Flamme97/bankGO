package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
)

type Storage interface {
	CreateAccount(*Account) error
	DeleteAccount(int) error
	UpdateAccount(*Account) error
	GetAccounts() ([]*Account, error)
	GetAccountByID(int) (*Account, error)
	GetAccountByNumber(int) (*Account, error)
}

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore() (*PostgresStore, error) {

	connStr := os.Getenv("DB_URL")
	fmt.Println(connStr)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &PostgresStore{
		db: db,
	}, nil
}

func (s *PostgresStore) Init() error {
	return s.createAccountTable()

}

func (s *PostgresStore) createAccountTable() error {
	query := `CREATE TABLE IF NOT EXISTS account (
	id serial primary key,
	firstName varchar(50),
	lastName  varchar(50),
	number		serial,
	balance 	serial, 	
	createdat timestamp,
	encrypted_password varchar(255)
	);`
	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStore) CreateAccount(a *Account) error {
	query := `
	INSERT INTO account 
	(firstName, lastName, number, balance, createdat, encrypted_password)
	VALUES($1, $2, $3, $4, $5, $6)
	`
	_, err := s.db.Query(query,
		a.FirstName,
		a.LastName,
		a.Number,
		a.Balance,
		a.CreatedAt,
		a.EncryptedPassword)
	if err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) UpdateAccount(*Account) error {
	return nil
}

func (s *PostgresStore) DeleteAccount(id int) error {
	_, err := s.db.Query("DELETE FROM account WHERE ID = $1", id)
	if err != nil {
		return err
	}

	return nil
}
func (s *PostgresStore) GetAccountByNumber(number int) (*Account, error) {
	rows, err := s.db.Query("SELECT * FROM account WHERE number = $1", number)
	if err != nil {
		return nil, err
	
	}
		defer rows.Close() // Always close rows after querying

	for rows.Next() {
		return scanIntoAccounts(rows)
	}
	return nil, fmt.Errorf("Account with number %d not found", number)

}

func (s *PostgresStore) GetAccountByID(id int) (*Account, error) {
	rows, err := s.db.Query("SELECT * FROM account WHERE ID = $1", id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		return scanIntoAccounts(rows)
	}
	return nil, fmt.Errorf("Account %d not found", id)
}

func (s *PostgresStore) GetAccounts() ([]*Account, error) {
	rows, err := s.db.Query("Select * from account")
	if err != nil {
		return nil, err
	}
	accounts := []*Account{}
	for rows.Next() {
		acc, err := scanIntoAccounts(rows)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, acc)
	}

	return accounts, nil
}

func scanIntoAccounts(rows *sql.Rows) (*Account, error) {

	acc := new(Account)
	err := rows.Scan(
		&acc.ID,
		&acc.FirstName,
		&acc.LastName,
		&acc.Number,
		&acc.Balance,
		&acc.CreatedAt,
		&acc.EncryptedPassword,
	)
	return acc, err
}
