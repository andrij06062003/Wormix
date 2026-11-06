package pvp

import (
	"errors"
	"net/http"

	"github.com/andrij06062003/Wormix/services/core/internal/api"
	"github.com/andrij06062003/Wormix/services/core/internal/csx"
	"github.com/andrij06062003/Wormix/services/core/internal/jsonx"
	"github.com/andrij06062003/Wormix/services/core/internal/proto"
)

// Calls to wormix-api's /api/internal/pvp/ routes. Each call acts as one player: the API checks the player's
// session key in X-SESSION-KEY.

// catchErr runs f, turning a panic (a C# exception) into an error.
func catchErr(f func()) error {
	if exc := csx.Catch(f); exc != nil {
		return exc
	}
	return nil
}

// getProfile is PvpApi.GetProfile: the player's own profile.
func getProfile(token string) (p proto.UserProfile, err error) {
	obj, err := api.PostPvP("pvp/profile", token, []byte("{}"))
	if err != nil {
		return p, err
	}
	err = catchErr(func() {
		data := obj.Get("data")
		if data == nil {
			csx.NullRef("data")
		}
		p = proto.DecodeUserProfile(data)
	})
	return p, err
}

// startBattle is PvpApi.StartBattle: records the battle for the player and takes the wager. Pass the reagents
// the first player got to the second player (nil makes the API choose them).
func startBattle(token string, battleID uint32, wager int32, reagents []int32) (p proto.UserProfile, got []int32, err error) {
	body := jsonx.NewObject().Int("BattleId", int64(battleID)).Int("Wager", int64(wager))
	if reagents == nil {
		body.Ints("Reagents", nil)
	} else {
		r := make([]int64, len(reagents))
		for i, x := range reagents {
			r[i] = int64(x)
		}
		body.Ints("Reagents", r)
	}
	obj, err := api.PostPvP("pvp/start_battle", token, body.Bytes())
	if err != nil {
		return p, nil, err
	}
	err = catchErr(func() {
		data := obj.Get("data")
		if data == nil {
			csx.NullRef("data")
		}
		if data.Kind != jsonx.Object {
			csx.Throw(csx.Generic, "Cannot access child value on %v", data.Kind)
		}
		ups := data.Obj.Get("UserProfileStructure")
		if ups == nil {
			csx.NullRef("UserProfileStructure")
		}
		p = proto.DecodeUserProfile(ups)
		got = []int32{}
		if r := data.Obj.Get("ReagentsForBattle"); r != nil {
			if l := proto.DecodeInts(r); l != nil {
				got = l
			}
		}
	})
	return p, got, err
}

// endBattle is PvpApi.EndBattle: pays out the result the PvP server decided: 1 win, 0 draw/cancelled, -1 loss.
func endBattle(token string, battleID uint32, result int) error {
	_, err := api.PostPvP("pvp/end_battle", token, jsonx.NewObject().Int("BattleId", int64(battleID)).Int("Result", int64(result)).Bytes())
	return err
}

// tryEndBattle is endBattle for when nobody waits for the answer: errors are only logged.
func tryEndBattle(token string, userID, battleID uint32, result int) {
	if err := endBattle(token, battleID, result); err != nil {
		logf("[PvP] can't record result %d of battle %d for %d: %v", result, battleID, userID, err)
	}
}

// isNotEnoughMoney: the API answers 422 when the player can't afford the wager.
func isNotEnoughMoney(err error) bool {
	var se *api.StatusError
	return errors.As(err, &se) && se.Status == http.StatusUnprocessableEntity
}
