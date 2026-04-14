-- Auth: users (approval gate). OTP flows use signed JWT tickets (no OTP storage in DB).

CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  is_approved BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Case-insensitive email uniqueness (expressions are not allowed in table UNIQUE(); use a unique index.)
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower ON users (lower(email));
