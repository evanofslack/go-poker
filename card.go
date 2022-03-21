package main

import "fmt"

type suit string

type denomination int

type card struct {
	suit  suit
	value denomination
}

func newCard(s suit, d denomination) *card {
	return &card{s, d}
}

func (c *card) String() string {
	return fmt.Sprint(c.value, " of ", c.suit)
}
