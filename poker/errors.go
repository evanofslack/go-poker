package poker

import (
	"errors"
)

//TODO: wrap more specific messages inside ErrIllegalAction

// ErrIllegalAction is returned when an Action is valid (all parameters are well-formed), but
// it is illegal per the laws of poker given the state of the game at that time. This could include
// illegal bet amounts, out-of-turn plays, or other violations of the rules.
var ErrIllegalAction = errors.New("this action cannot be performed at this time")
var ErrOutOfBounds = errors.New("error adding player at position (out of bounds)")
var ErrInvalidPosition = errors.New("error assigning player position, position already taken")
var ErrStartGame = errors.New("error starting the game, one or more players not ready")

/*
var ErrBuyTooBig = errors.New("this would exceed the maximum configured purchased stack size")
var ErrNotEnoughPlayers = errors.New("need more players to start the round")
*/

var errInternalBadGameStage = errors.New("internal error: bad game stage")
