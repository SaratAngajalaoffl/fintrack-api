-- OTP challenges moved to stateless signed JWTs; drop legacy table if present.

DROP TABLE IF EXISTS otp_challenges;
