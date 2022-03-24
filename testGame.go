package main

import (
	"fmt"

	"github.com/alexclewontin/riverboat"
)

func testGame() {

	g := riverboat.NewGame()
	p1 := g.AddPlayer()
	p2 := g.AddPlayer()
	p3 := g.AddPlayer()

	players := []uint{p1, p2, p3}

	for _, p := range players {
		err := riverboat.BuyIn(g, p, 1000)
		if err != nil {
			fmt.Println(err)
		}
		err = riverboat.ToggleReady(g, p, 0)
		if err != nil {
			fmt.Println(err)
		}
	}

	godView := g.GenerateOmniView()
	dealer := godView.DealerNum
	err := riverboat.Deal(g, dealer, 0)
	if err != nil {
		fmt.Println(err)
	}

	godView = g.GenerateOmniView()
	action := godView.ActionNum
	fmt.Println(action)
	err = riverboat.Bet(g, action, 25)
	if err != nil {
		fmt.Println(err)
	}

	godView = g.GenerateOmniView()
	action = godView.ActionNum
	fmt.Println(action)
	err = riverboat.Bet(g, action, 15)
	if err != nil {
		fmt.Println(err)
	}
	godView = g.GenerateOmniView()
	action = godView.ActionNum
	fmt.Println(action)
	err = riverboat.Bet(g, action, 0)
	if err != nil {
		fmt.Println(err)
	}

	for _, i := range []uint{1, 2, 0} {
		err = riverboat.Bet(g, i, 0)
		if err != nil {
			fmt.Println(err)
		}
	}

	for _, i := range []uint{1, 2, 0} {
		err = riverboat.Bet(g, i, 0)
		if err != nil {
			fmt.Println(err)
		}
	}

	for _, i := range []uint{1, 2, 0} {
		err = riverboat.Bet(g, i, 0)
		if err != nil {
			fmt.Println(err)
		}
	}
	godView = g.GenerateOmniView()
	fmt.Printf("%+v\n", godView)
}
