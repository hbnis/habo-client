package client

import (
	"github.com/ao-data/albiondata-client/lib"
	"github.com/ao-data/albiondata-client/log"
)

type operationJoinResponse struct {
	CharacterID   lib.CharacterID `mapstructure:"1"`
	CharacterName string          `mapstructure:"2"`
	Location      string          `mapstructure:"8"`
	GuildID       lib.CharacterID `mapstructure:"56"`
	GuildName     string          `mapstructure:"58"`
}

//CharacterPartsJSON string          `mapstructure:"6"`
//Edition            string          `mapstructure:"38"`

func (op operationJoinResponse) Process(state *albionState) {
	log.Debugf("Got JoinResponse operation...")

	// Keep the last valid Albion server ID. The packet listener refreshes it
	// from the source IP on every game packet, including real server switches.
	// Clearing it here created a short window where valid market uploads were
	// sent with server ID 0 and rejected by The Habo Hub.

	location := normalizeLocationID(op.Location)
	if location != "" {
		log.Infof("Updating player location to %v.", location)
		state.LocationId = location
	} else {
		log.Debugf("Ignoring implausible join location value: %q", op.Location)
	}

	if state.CharacterId != op.CharacterID {
		log.Infof("Updating player ID to %v.", op.CharacterID)
	}
	state.CharacterId = op.CharacterID

	if state.CharacterName != op.CharacterName {
		log.Infof("Updating player to %v.", op.CharacterName)
	}
	state.CharacterName = op.CharacterName
}
