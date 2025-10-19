package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// GetBlockList retrieves the blocklist for a given user PID.
func GetBlockList(manager *common_globals.MatchmakingManager, userPID types.PID) ([]uint64, *nex.Error) {
	rows, err := manager.Database.Query(`
		SELECT blocked_pid FROM matchmaking.block_lists WHERE user_pid = $1
	`, uint64(userPID))

	if err != nil {
		return nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}
	defer rows.Close()

	var blockList []uint64
	for rows.Next() {
		var blockedPID uint64
		if err := rows.Scan(&blockedPID); err != nil {
			return nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
		blockList = append(blockList, blockedPID)
	}

	if err := rows.Err(); err != nil {
		return nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return blockList, nil
}
