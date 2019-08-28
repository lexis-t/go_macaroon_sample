package macaroon_proof

import (
	"context"
	"fmt"
	"gopkg.in/check.v1"
	"gopkg.in/macaroon-bakery.v2/bakery"
	"gopkg.in/macaroon-bakery.v2/bakery/checkers"
	"gopkg.in/macaroon.v2"
	"testing"
	"time"
)

func Test(t *testing.T) {
	check.TestingT(t)
	//check.RunAll(&check.RunConf{
	//	Output:        os.Stdout,
	//	Verbose:       true,
	//})
}

func testCavCheck(ctx context.Context, cond, arg string) error {
	if arg != "12345" {
		return fmt.Errorf("%s condition %s does not match 12345", cond, arg)
	}
	return nil
}

type MacaroonSuite struct {
	ctx context.Context
	key *bakery.KeyPair
	permissions []bakery.Op
	justPast time.Time
	future time.Time
}

var _ = check.Suite(&MacaroonSuite{})

func (s *MacaroonSuite) SetUpSuite(c *check.C) {
	s.ctx = context.Background()
	s.key = bakery.MustGenerateKey()
	s.permissions = []bakery.Op {{Entity: "test", Action: "testAction"}}
	s.justPast = time.Now()
	s.future = s.justPast.Add(time.Hour)

}

func (s *MacaroonSuite) TestSimpleMacaroon(c *check.C) {

	params := bakery.BakeryParams{
		Key:              s.key,
		Location:         "macaroon_proof",
	}

	b := bakery.New(params)

	m, e := b.Oven.NewMacaroon(s.ctx, bakery.Version2, nil, s.permissions...)
	c.Assert(e, check.IsNil)
	c.Logf("Macaroon: %+v", m)

	mData, e := m.MarshalJSON()
	c.Assert(e, check.IsNil)

	mDecoded := &macaroon.Macaroon{}
	e = mDecoded.UnmarshalJSON(mData)
	c.Assert(e, check.IsNil)

	ch := b.Checker.Auth(macaroon.Slice{mDecoded})

	_, e = ch.Allow(s.ctx, s.permissions...)
	c.Assert(e, check.IsNil)
}

func (s *MacaroonSuite) TestSimpleMacaroonMoreOps(c *check.C) {

	params := bakery.BakeryParams{
		Key:              s.key,
		Location:         "macaroon_proof",
	}

	b := bakery.New(params)

	m, e := b.Oven.NewMacaroon(s.ctx, bakery.Version2, nil, []bakery.Op {{Entity: "test", Action: "testAction"}, {Entity: "test", Action: "wrongAction"}}...)
	c.Assert(e, check.IsNil)
	c.Logf("Macaroon: %+v", m)

	mData, e := m.MarshalJSON()
	c.Assert(e, check.IsNil)

	mDecoded := &macaroon.Macaroon{}
	e = mDecoded.UnmarshalJSON(mData)
	c.Assert(e, check.IsNil)

	ch := b.Checker.Auth(macaroon.Slice{mDecoded})

	_, e = ch.Allow(s.ctx, s.permissions...)
	c.Assert(e, check.IsNil)
}

func (s *MacaroonSuite) TestSimpleMacaroonWrongOp(c *check.C) {

	params := bakery.BakeryParams{
		Key:              s.key,
		Location:         "macaroon_proof",
	}

	b := bakery.New(params)

	m, e := b.Oven.NewMacaroon(s.ctx, bakery.Version2, nil, []bakery.Op {{Entity: "test", Action: "wrongAction"}}...)
	c.Assert(e, check.IsNil)
	c.Logf("Macaroon: %+v", m)

	mData, e := m.MarshalJSON()
	c.Assert(e, check.IsNil)

	mDecoded := &macaroon.Macaroon{}
	e = mDecoded.UnmarshalJSON(mData)
	c.Assert(e, check.IsNil)

	ch := b.Checker.Auth(macaroon.Slice{mDecoded})

	_, e = ch.Allow(s.ctx, s.permissions...)
	c.Assert(e, check.NotNil)
}

func (s *MacaroonSuite) TestMacaroonWithTimeCaveat(c *check.C) {

	params := bakery.BakeryParams{
		Key:              s.key,
		Location:         "macaroon_proof",
	}

	b := bakery.New(params)

	m, e := b.Oven.NewMacaroon(s.ctx, bakery.Version2, []checkers.Caveat{
		checkers.TimeBeforeCaveat(s.future),
	}, s.permissions...)

	c.Assert(e, check.IsNil)
	c.Logf("Macaroon: %+v", m)

	mData, e := m.MarshalJSON()
	c.Assert(e, check.IsNil)

	mDecoded := &macaroon.Macaroon{}
	e = mDecoded.UnmarshalJSON(mData)
	c.Assert(e, check.IsNil)

	ch := b.Checker.Auth(macaroon.Slice{mDecoded})

	_, e = ch.Allow(s.ctx, s.permissions...)
	c.Assert(e, check.IsNil)
}

func (s *MacaroonSuite) TestMacaroonWithPastTimeCaveat(c *check.C) {

	//checker := checkers.Checker{}
	//checker.Register()

	params := bakery.BakeryParams{
		Checker:          nil,
		Key:              s.key,
		Location:         "macaroon_proof",
	}

	b := bakery.New(params)

	m, e := b.Oven.NewMacaroon(s.ctx, bakery.Version2, []checkers.Caveat{
		checkers.TimeBeforeCaveat(s.justPast),
	}, s.permissions...)

	c.Assert(e, check.IsNil)

	mData, e := m.MarshalJSON()
	c.Assert(e, check.IsNil)

	mDecoded := &macaroon.Macaroon{}
	e = mDecoded.UnmarshalJSON(mData)
	c.Assert(e, check.IsNil)
	c.Logf("Macaroon: %+v", mDecoded)

	ch := b.Checker.Auth(macaroon.Slice{mDecoded})
	_, e = ch.Allow(s.ctx, s.permissions...)
	c.Assert(e, check.NotNil)
}

func (s *MacaroonSuite) TestMacaroonWithCustomCaveat(c *check.C) {

	checker := func() *checkers.Checker {
		c := checkers.New(nil)
		c.Namespace().Register("prooftest", "")
		c.Register("testname", "prooftest", testCavCheck)
		return c
	}()

	params := bakery.BakeryParams {
		Checker:  checker,
		Key:      s.key,
		Location: "macaroon_proof",
	}

	b := bakery.New(params)

	m, e := b.Oven.NewMacaroon(s.ctx, bakery.Version2, []checkers.Caveat{
		{
			Condition: checkers.Condition("testname", "12345"),
			Namespace: "prooftest",
		},
	}, s.permissions...)

	c.Assert(e, check.IsNil)

	mData, e := m.MarshalJSON()
	c.Assert(e, check.IsNil)

	mDecoded := &macaroon.Macaroon{}
	e = mDecoded.UnmarshalJSON(mData)
	c.Assert(e, check.IsNil)
	c.Logf("Macaroon: %+v", mDecoded)

	ch := b.Checker.Auth(macaroon.Slice{mDecoded})

	_, e = ch.Allow(s.ctx, s.permissions...)
	c.Assert(e, check.IsNil)
}

func (s *MacaroonSuite) TestMacaroonWithCustomCaveatFails(c *check.C) {

	checker := func() *checkers.Checker {
		c := checkers.New(nil)
		c.Namespace().Register("prooftest", "")
		c.Register("testname", "prooftest", testCavCheck)
		return c
	}()

	params := bakery.BakeryParams {
		Checker:  checker,
		Key:      s.key,
		Location: "macaroon_proof",
	}

	b := bakery.New(params)

	m, e := b.Oven.NewMacaroon(s.ctx, bakery.Version2, []checkers.Caveat{
		{
			Condition: checkers.Condition("testname", "123456"),
			Namespace: "prooftest",
		},
	}, s.permissions...)

	c.Assert(e, check.IsNil)

	mData, e := m.MarshalJSON()
	c.Assert(e, check.IsNil)

	mDecoded := &macaroon.Macaroon{}
	e = mDecoded.UnmarshalJSON(mData)
	c.Assert(e, check.IsNil)
	c.Logf("Macaroon: %+v", mDecoded)

	ch := b.Checker.Auth(macaroon.Slice{mDecoded})

	_, e = ch.Allow(s.ctx, s.permissions...)
	c.Assert(e, check.NotNil)
}
