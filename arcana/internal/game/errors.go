package game

// Error is a rejected action. Code is stable and sent to clients.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

var (
	ErrUnknownPlayer     = &Error{"not_in_match", "player is not in this match"}
	ErrMatchFinished     = &Error{"match_finished", "the match is over"}
	ErrWrongPhase        = &Error{"wrong_phase", "not allowed in this phase"}
	ErrNotYourTurn       = &Error{"not_your_turn", "it is not your turn"}
	ErrCardNotInHand     = &Error{"card_not_in_hand", "that card is not in your hand"}
	ErrNotEnoughMana     = &Error{"not_enough_mana", "not enough mana"}
	ErrInvalidTarget     = &Error{"invalid_target", "that target is not allowed"}
	ErrAbilityUsed       = &Error{"ability_used", "the hero ability was already used this turn"}
	ErrUltimateNotReady  = &Error{"ultimate_not_ready", "the ultimate is not charged"}
	ErrAlreadyMulliganed = &Error{"already_mulliganed", "the mulligan was already taken"}
	ErrTooManyCards      = &Error{"too_many_cards", "too many cards to replace"}
	ErrNothingToRepeat   = &Error{"nothing_to_repeat", "no previous card to repeat"}
	ErrHandFull          = &Error{"hand_full", "the hand is full"}
)
