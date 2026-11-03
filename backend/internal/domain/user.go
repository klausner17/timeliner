package domain

import "time"

// Tint é a cor do avatar: uma das cores de trilha ou neutral.
type Tint string

const TintNeutral Tint = "neutral"

func (t Tint) Valid() bool { return t == TintNeutral || TrackColor(t).Valid() }

type User struct {
	ID           string
	Handle       string
	Name         string
	Email        string
	PasswordHash string
	Bio          string
	Tint         Tint
	CreatedAt    time.Time
	// EmailVerifiedAt é quando a pessoa abriu o link de confirmação; nil enquanto não confirmou.
	EmailVerifiedAt *time.Time
}

// UserStats são os números do perfil.
type UserStats struct {
	Timelines             int
	Events                int
	ApprovedContributions int
	Followers             int
	// Following diz se quem está vendo segue este usuário.
	Following bool
}

// Session é um login ativo. Só o hash do token fica guardado.
type Session struct {
	TokenHash string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// TokenPurpose diz para que serve um AuthToken.
type TokenPurpose string

const (
	TokenVerifyEmail   TokenPurpose = "verificar_email"
	TokenResetPassword TokenPurpose = "redefinir_senha"
)

// AuthToken é um link de uso único mandado por e-mail. Só o hash fica guardado.
type AuthToken struct {
	TokenHash string
	UserID    string
	Purpose   TokenPurpose
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Email é uma mensagem para mandar a alguém.
type Email struct {
	To      string
	Subject string
	Text    string
	HTML    string
}
