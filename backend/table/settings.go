package table

import (
	"errors"
	"fmt"
	"strings"

	"github.com/evanofslack/go-poker/holdem"
)

// Settings are picked by the host when creating a table. The host can change
// them later, all but the seat count.
// Buy-ins are in chips, times in milliseconds.
type Settings struct {
	Seats         uint `json:"seats"`
	SmallBlind    uint `json:"sb"`
	BigBlind      uint `json:"bb"`
	MinBuy        uint `json:"minBuy"`
	MaxBuy        uint `json:"maxBuy"`
	TurnTime      uint `json:"turnTime"`      // time to act before checking or folding for you, 0 for no limit
	NextHandDelay uint `json:"nextHandDelay"` // pause after a hand before dealing the next
	DecisionTime  uint `json:"decisionTime"`  // to answer a straddle ask or a run-it vote
	Public        bool `json:"public"`        // listed on the home page
	ApproveRebuys bool `json:"approveRebuys"` // rebuys, and sitting back down, wait for the host

	Game  string `json:"game"`  // GameHoldem, GameOmaha or GameOmaha5
	Limit string `json:"limit"` // LimitNo or LimitPot
	Deck  string `json:"deck"`  // DeckFull or DeckShort

	// house rules
	Ante     uint `json:"ante"`     // from every player each hand, 0 for none
	DuckHunt bool `json:"duckHunt"` // players can turn up the rest of the board after a hand ends early
	// SevenDeuce is the bounty everyone dealt in pays a player who wins the
	// main pot with the worst hand, 7-2, or J-6 on a short deck, and shows
	// it, 0 for none. Hold'em only.
	SevenDeuce uint   `json:"sevenDeuce"`
	Straddle   string `json:"straddle"` // StraddleOff, StraddleOptional or StraddleForced
	RunIt      bool   `json:"runIt"`    // once everyone left is all in, or all but one, they vote to run the rest of the board out more than once

	// bomb pots, on one board or two, called by the host or on a schedule
	BombPot      uint   `json:"bombPot"`      // what everyone puts in a bomb pot, 0 for no bomb pots
	Special      string `json:"special"`      // which bomb pot: SpecialBomb or SpecialDoubleBomb
	SpecialEvery uint   `json:"specialEvery"` // one every this many hands or orbits, 0 for only when the host calls one
	SpecialUnit  string `json:"specialUnit"`  // UnitHands or UnitOrbits
}

// Special hands, and the units of their schedule
const (
	SpecialBomb       = "bomb"
	SpecialDoubleBomb = "double-bomb"
	UnitHands         = "hands"
	UnitOrbits        = "orbits"
)

// specialNames say what each special hand is.
var specialNames = map[string]string{
	SpecialBomb:       "bomb pot",
	SpecialDoubleBomb: "double board bomb pot",
}

func isDouble(kind string) bool { return kind == SpecialDoubleBomb }

// scheduled is whether set deals special hands on its own.
func (set Settings) scheduled() bool { return set.SpecialEvery > 0 }

// fitsDouble is whether the deck has two boards' worth for every seat.
func (set Settings) fitsDouble() bool {
	return int(set.Seats)*holeCards[set.Game]+10 <= len(holdem.Deck(set.config().Game))
}

// Games and limits
const (
	GameHoldem = "holdem"
	GameOmaha  = "omaha"
	GameOmaha5 = "omaha5"
	LimitNo    = "no"
	LimitPot   = "pot"
	DeckFull   = "full"
	DeckShort  = "short" // 6+: 2s to 5s out
)

// holeCards is how many hole cards each game deals.
var holeCards = map[string]int{GameHoldem: 2, GameOmaha: 4, GameOmaha5: 5}

// config is the game's config for set.
func (set Settings) config() holdem.Config {
	return holdem.Config{
		Game:       holdem.Game{Hole: holeCards[set.Game], Pot: set.Limit == LimitPot, Short: set.Deck == DeckShort},
		SmallBlind: set.SmallBlind,
		BigBlind:   set.BigBlind,
		Ante:       set.Ante,
	}
}

// GameName names set's game the short way, like "NLH", "PLO5" or "6+ NLH".
func GameName(set Settings) string {
	name := map[string]string{GameHoldem: "H", GameOmaha: "O", GameOmaha5: "O5"}[set.Game]
	if set.Limit == LimitPot {
		name = "PL" + name
	} else {
		name = "NL" + name
	}
	if set.Deck == DeckShort {
		return "6+ " + name
	}
	return name
}

// Straddle settings. Forced posts under the gun's straddle for them;
// re-straddles stay optional.
const (
	StraddleOff      = "off"
	StraddleOptional = "optional"
	StraddleForced   = "forced"
)

// DefaultSettings are used for anything create-table leaves out.
var DefaultSettings = Settings{
	Seats:         8,
	SmallBlind:    10,
	BigBlind:      20,
	MinBuy:        1000,
	MaxBuy:        4000,
	TurnTime:      30000,
	NextHandDelay: 5000,
	DecisionTime:  5000,
	DuckHunt:      true,
	RunIt:         true,
	Straddle:      StraddleOff,
	Game:          GameHoldem,
	Limit:         LimitNo,
	Deck:          DeckFull,
	Special:       SpecialBomb,
	SpecialUnit:   UnitHands,
}

// settings limits, mirrored by the create table form
const (
	maxSeats         = 9
	maxBigBlind      = 1_000_000
	maxBuyInBB       = 1000 // max buy-in, in big blinds
	minTurnTime      = 10_000
	maxTurnTime      = 300_000
	minNextHandDelay = 2000
	maxNextHandDelay = 15_000
	minDecisionTime  = 2000
	maxDecisionTime  = 30_000
	maxSevenDeuceBB  = 10  // the most a 7-2 bounty can be, in big blinds
	maxBombBB        = 20  // the most a bomb pot can be, in big blinds
	maxEveryHands    = 100 // the most hands between scheduled special hands
	maxEveryOrbits   = 20
)

// Validate returns why s can't be used for a new table, if it can't.
func (s Settings) Validate() error {
	switch {
	case s.Seats < 2 || s.Seats > maxSeats:
		return fmt.Errorf("seats must be between 2 and %d", maxSeats)
	case s.BigBlind < 1:
		return errors.New("big blind must be at least 1")
	case s.SmallBlind > s.BigBlind:
		return errors.New("small blind can't be more than the big blind")
	case s.BigBlind > maxBigBlind:
		return fmt.Errorf("big blind can't be more than %d", maxBigBlind)
	case s.MinBuy < s.BigBlind || s.MinBuy > s.MaxBuy:
		return errors.New("min buy-in must be between 1 big blind and the max buy-in")
	case s.MaxBuy > maxBuyInBB*s.BigBlind:
		return fmt.Errorf("max buy-in can't be more than %d big blinds", maxBuyInBB)
	case s.Ante > s.BigBlind:
		return errors.New("ante can't be more than the big blind")
	case s.SevenDeuce > maxSevenDeuceBB*s.BigBlind:
		return fmt.Errorf("7-2 bounty can't be more than %d big blinds", maxSevenDeuceBB)
	case s.TurnTime != 0 && (s.TurnTime < minTurnTime || s.TurnTime > maxTurnTime):
		return fmt.Errorf("turn timer must be off or between %d and %d seconds", minTurnTime/1000, maxTurnTime/1000)
	case s.NextHandDelay < minNextHandDelay || s.NextHandDelay > maxNextHandDelay:
		return fmt.Errorf("next hand delay must be between %d and %d seconds", minNextHandDelay/1000, maxNextHandDelay/1000)
	case s.DecisionTime < minDecisionTime || s.DecisionTime > maxDecisionTime:
		return fmt.Errorf("decision time must be between %d and %d seconds", minDecisionTime/1000, maxDecisionTime/1000)
	case s.Straddle != StraddleOff && s.Straddle != StraddleOptional && s.Straddle != StraddleForced:
		return errors.New("straddle must be off, optional or forced")
	case holeCards[s.Game] == 0:
		return errors.New("game must be holdem, omaha or omaha5")
	case s.Limit != LimitNo && s.Limit != LimitPot:
		return errors.New("limit must be no or pot")
	case s.Deck != DeckFull && s.Deck != DeckShort:
		return errors.New("deck must be full or short")
	case int(s.Seats)*holeCards[s.Game]+5 > len(holdem.Deck(s.config().Game)):
		return fmt.Errorf("%s deals too many cards for %d seats", GameName(s), s.Seats)
	case s.BombPot > maxBombBB*s.BigBlind:
		return fmt.Errorf("bomb pot can't be more than %d big blinds", maxBombBB)
	case specialNames[s.Special] == "":
		return errors.New("special hand must be bomb or double-bomb")
	case s.SpecialUnit != UnitHands && s.SpecialUnit != UnitOrbits:
		return errors.New("the special hand schedule must be in hands or orbits")
	case s.SpecialUnit == UnitHands && s.SpecialEvery > maxEveryHands:
		return fmt.Errorf("special hands can't be further apart than %d hands", maxEveryHands)
	case s.SpecialUnit == UnitOrbits && s.SpecialEvery > maxEveryOrbits:
		return fmt.Errorf("special hands can't be further apart than %d orbits", maxEveryOrbits)
	case s.scheduled() && s.BombPot == 0:
		return errors.New("turn on bomb pots to schedule them")
	case s.scheduled() && isDouble(s.Special) && !s.fitsDouble():
		return fmt.Errorf("%s deals too many cards for two boards at %d seats", GameName(s), s.Seats)
	case s.SevenDeuce > 0 && holeCards[s.Game] != 2:
		return errors.New("the worst hand bounty is for hold'em")
	}
	return nil
}

// updateSettings changes the settings now between hands, or from the next
// hand mid-hand.
func (r *reducer) updateSettings(p *Person, set Settings) error {
	s := r.s
	if p.Name != s.Host {
		return errNotHost
	}
	set.Seats = s.Settings.Seats
	if err := set.Validate(); err != nil {
		return fail("%s", err)
	}
	if !s.Game.InHand() {
		r.applySettings(p.Name+" changed", set)
		return nil
	}
	if set == s.Settings {
		if s.Next != nil {
			s.Next = nil
			r.log("%s undid the settings changes for the next hand", p.Name)
		}
		return nil
	}
	s.Next = &set
	// some changes, like the unit of an unused schedule, aren't worth a line
	if changes := settingsChanges(s.Settings, set); changes != "" {
		r.log("%s changed %s, from the next hand", p.Name, changes)
	}
	return nil
}

// applySettings switches to set between hands, logging what changed after who.
func (r *reducer) applySettings(who string, set Settings) {
	s := r.s
	changes := settingsChanges(s.Settings, set)
	if s.Settings.SmallBlind != set.SmallBlind || s.Settings.BigBlind != set.BigBlind || s.Settings.Ante != set.Ante {
		r.dropStraddles() // asked at the old blinds
	}
	s.Settings, s.Next = set, nil
	if set.BombPot == 0 {
		s.SpecialNext = ""
	}
	if !set.scheduled() {
		s.SpecialHands = 0
	}
	s.Game.SetConfig(set.config())
	if changes != "" {
		r.log("%s %s", who, changes)
	}
	r.scheduleDeal()
}

// settingsChanges lists what changed from old to set, like "the blinds to
// 25/50, the turn timer to 60s", or "" for nothing.
func settingsChanges(old, set Settings) string {
	var c []string
	add := func(changed bool, format string, args ...any) {
		if changed {
			c = append(c, fmt.Sprintf(format, args...))
		}
	}
	onOff := func(on bool) string {
		if on {
			return "on"
		}
		return "off"
	}
	chips := func(n uint) string {
		if n == 0 {
			return "off"
		}
		return fmt.Sprint(n)
	}
	add(old.Game != set.Game || old.Limit != set.Limit || old.Deck != set.Deck, "the game to %s", GameName(set))
	add(old.SmallBlind != set.SmallBlind || old.BigBlind != set.BigBlind, "the blinds to %d/%d", set.SmallBlind, set.BigBlind)
	add(old.Ante != set.Ante, "the ante to %s", chips(set.Ante))
	add(old.MinBuy != set.MinBuy || old.MaxBuy != set.MaxBuy, "the buy-in to %d-%d", set.MinBuy, set.MaxBuy)
	if set.TurnTime == 0 {
		add(old.TurnTime != 0, "the turn timer off")
	} else {
		add(old.TurnTime != set.TurnTime, "the turn timer to %ds", set.TurnTime/1000)
	}
	add(old.NextHandDelay != set.NextHandDelay, "the time between hands to %ds", set.NextHandDelay/1000)
	add(old.Public != set.Public, "the table to %s", visibility(set.Public))
	add(old.ApproveRebuys != set.ApproveRebuys, "rebuy approval %s", onOff(set.ApproveRebuys))
	add(old.DuckHunt != set.DuckHunt, "duck hunt %s", onOff(set.DuckHunt))
	add(old.SevenDeuce != set.SevenDeuce, "the %s bounty to %s", bountyKind(set.config().Game), chips(set.SevenDeuce))
	add(old.Straddle != set.Straddle, "straddles %s", set.Straddle)
	add(old.RunIt != set.RunIt, "run it %s", onOff(set.RunIt))
	add(old.DecisionTime != set.DecisionTime, "the decision time to %ds", set.DecisionTime/1000)
	add(old.BombPot != set.BombPot, "bomb pots to %s", chips(set.BombPot))
	if !set.scheduled() {
		add(old.scheduled(), "bomb pots only when called")
		add(set.BombPot > 0 && old.Special != set.Special, "double board bomb pots %s", onOff(isDouble(set.Special)))
	} else {
		add(old.Special != set.Special || old.SpecialEvery != set.SpecialEvery || old.SpecialUnit != set.SpecialUnit,
			"a %s every %s", specialNames[set.Special], every(set.SpecialEvery, set.SpecialUnit))
	}
	return strings.Join(c, ", ")
}

// every says how often, like "3 hands" or "orbit".
func every(n uint, unit string) string {
	if n == 1 {
		return strings.TrimSuffix(unit, "s")
	}
	return fmt.Sprintf("%d %s", n, unit)
}

func visibility(public bool) string {
	if public {
		return "public"
	}
	return "private"
}

func (r *reducer) setPassword(p *Person, password string) error {
	s := r.s
	if p.Name != s.Host {
		return errNotHost
	}
	if password == "" {
		s.Password = ""
		r.log("%s removed the password", p.Name)
	} else {
		s.Password = hashPassword(password)
		r.log("%s changed the password", p.Name)
	}
	r.send("", NewPassword{password})
	return nil
}
