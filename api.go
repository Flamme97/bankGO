package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	jwt "github.com/golang-jwt/jwt/v4"
)

type APIServer struct {
	listenAddr string
	store      Storage
}
type apiFunc func(http.ResponseWriter, *http.Request) error

type APIError struct {
	Error string `json:"error"`
}

func NewAPIServer(listenAddr string, store Storage) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		store:      store,
	}
}

func (s *APIServer) Run() {
	router := chi.NewMux()

	router.HandleFunc("/login", makeHTTPHandleFunc(s.HandleLogin))
	router.HandleFunc("/account", makeHTTPHandleFunc(s.HandleAccount))

	router.HandleFunc("/account/{id}", withJWTAuth(makeHTTPHandleFunc(s.HandleSingleAccount), s.store))
	router.HandleFunc("/transfer", makeHTTPHandleFunc(s.HandleTransfer))

	log.Println("Server API running on port", s.listenAddr)
	http.ListenAndServe(s.listenAddr, router)
}

func (s *APIServer) HandleAccount(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return s.HandleGetAccount(w, r)
	}
	if r.Method == "POST" {
		return s.HandleCreateAccount(w, r)
	}

	return fmt.Errorf("methods not allowed %s", r.Method)
}

// 30264
func (s *APIServer) HandleLogin(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		PermissionsDenied(w)
		return nil
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}

	acc, err := s.store.GetAccountByNumber(int(req.Number))
	if err != nil {
		return err // handle this reponse as json
	}

	if !acc.ValidatePW(req.Password) {
		return fmt.Errorf("failed to login")
	}

	token, err := createJWT(acc)
	if err != nil {
		return err
	}

	resp := LoginReponse{
		Token:  token,
		Number: acc.Number,
	}

	fmt.Printf("%+v\n", acc)
	return WriteJSON(w, http.StatusOK, resp)

}

func (s *APIServer) HandleTransfer(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return s.HandleGetAccount(w, r)
	}
	if r.Method == "POST" {
		return s.HandleTransferToAccount(w, r)
	}
	if r.Method == "DELETE" {
		return s.HandleDeleteAccount(w, r)
	}

	return fmt.Errorf("methods not allowed %s", r.Method)
}

func (s *APIServer) HandleSingleAccount(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return s.HandleGetAccountByID(w, r)
	}
	if r.Method == "DELETE" {
		return s.HandleDeleteAccount(w, r)
	}
	return fmt.Errorf("method not provided")

}

func (s *APIServer) HandleGetAccount(w http.ResponseWriter, r *http.Request) error {
	accounts, err := s.store.GetAccounts()
	if err != nil {
		return err
	}
	return WriteJSON(w, http.StatusOK, accounts)
}

func (s *APIServer) HandleGetAccountByID(w http.ResponseWriter, r *http.Request) error {
	id, err := getID(r)
	if err != nil {
		return err
	}
	account, err := s.store.GetAccountByID(id)
	if err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, account)
}

func (s *APIServer) HandleCreateAccount(w http.ResponseWriter, r *http.Request) error {
	createAccountReq := new(CreateAccountRequest)
	if err := json.NewDecoder(r.Body).Decode(createAccountReq); err != nil {
		return err
	}
	account, _ := NewAccount(createAccountReq.FirstName, createAccountReq.LastName, createAccountReq.Password)
	if err := s.store.CreateAccount(account); err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, account)
}

func (s *APIServer) HandleDeleteAccount(w http.ResponseWriter, r *http.Request) error {
	id, err := getID(r)
	if err != nil {
		return err
	}
	if err := s.store.DeleteAccount(id); err != nil {
		return err
	}
	return WriteJSON(w, http.StatusOK, map[string]int{"deleted:": id})
}

func (s *APIServer) HandleTransferToAccount(w http.ResponseWriter, r *http.Request) error {
	transferReq := new(TransferRequest)
	if err := json.NewDecoder(r.Body).Decode(transferReq); err != nil {
		return err
	}

	defer r.Body.Close()
	return WriteJSON(w, http.StatusOK, transferReq)
}

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			WriteJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
		}
	}

}

func getID(r *http.Request) (int, error) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return id, fmt.Errorf("invalid ID provided %v", idStr)
	}
	return id, nil
}

func createJWT(account *Account) (string, error) {
	// Create the Claims
	claims := &jwt.MapClaims{
		"expiresAt":     15000,
		"accountNumber": account.Number,
	}
	secret := os.Getenv("JWT_SECRET")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

func PermissionsDenied(w http.ResponseWriter) {
	WriteJSON(w, http.StatusForbidden, APIError{Error: "Permission denied"})

}

func withJWTAuth(handlerFunc http.HandlerFunc, s Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("calling JWT auth middlware")

		tokenString := r.Header.Get("x-jwt-token")
		if tokenString == "" {
			PermissionsDenied(w)
			return
		}
		token, err := validateJWT(tokenString)
		if err != nil {
			PermissionsDenied(w)
			return
		}
		userID, err := getID(r)
		if err != nil {
			PermissionsDenied(w)
			return
		}

		account, err := s.GetAccountByID(userID)
		if err != nil {
			PermissionsDenied(w)
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		if account.Number != int64(claims["accountNumber"].(float64)) {
			PermissionsDenied(w)
			return
		}

		fmt.Println(account)
		fmt.Println(token)
		handlerFunc(w, r)
	}

}

func validateJWT(tokenStr string) (*jwt.Token, error) {
	secret := os.Getenv("JWT_SECRET")
	return jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {

		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(secret), nil
	})
}
