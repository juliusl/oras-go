package remotes

type HandleAuthChallenge = func(AuthChallengeError) (*Registry, error)
