package auth

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserView struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	Status string `json:"status"`
}

type TokenResponse struct {
	AccessToken string   `json:"accessToken"`
	User        UserView `json:"user"`
}
