package auth

const (
	// SessionCookie matches web/src/lib/auth/constants SESSION_COOKIE.
	SessionCookie = "fintrack_session"
	OTPLength       = 6
	OTPTTL          = 15 * 60 * 1000 // milliseconds, matches TS OTP_TTL_MS
	JWT_TTL_SEC     = 7 * 24 * 60 * 60
	// PasswordBcryptCost matches bcryptjs ROUNDS in password.ts.
	PasswordBcryptCost = 12
	// OTPBcryptCost matches OTP_BCRYPT_ROUNDS in otp.ts.
	OTPBcryptCost = 10
)
