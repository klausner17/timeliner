-- +goose Up
-- Cadastro aberto: confirmação de e-mail e links de uso único (confirmar e-mail, redefinir senha).
ALTER TABLE users ADD COLUMN email_verified_at timestamptz;
-- Quem já tinha conta foi criado à mão pelo create-user: conta como confirmado.
UPDATE users SET email_verified_at = created_at;

CREATE TABLE auth_tokens (
    token_hash text PRIMARY KEY,
    user_id    text NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    purpose    text NOT NULL,
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL
);
CREATE INDEX auth_tokens_user_idx ON auth_tokens (user_id, purpose);

-- +goose Down
DROP TABLE auth_tokens;
ALTER TABLE users DROP COLUMN email_verified_at;
