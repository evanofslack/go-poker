package table

// bombPot makes the next hand a bomb pot, or calls it off. Mid-hand it's the
// hand after this one.
func (r *reducer) bombPot(p *Person, on bool) error {
	s := r.s
	switch {
	case p.Name != s.Host:
		return errNotHost
	case on && s.Settings.BombPot == 0:
		return fail("set a bomb pot amount first")
	case on && s.BombNext:
		return fail("the next hand is already a bomb pot")
	case !on && !s.BombNext:
		return fail("the next hand isn't a bomb pot")
	}
	s.BombNext = on
	if on {
		r.log("%s makes the next hand a bomb pot of %d each", p.Name, s.Settings.BombPot)
	} else {
		r.log("%s called off the bomb pot", p.Name)
	}
	return nil
}
