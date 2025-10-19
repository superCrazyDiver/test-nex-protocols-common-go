package matchmake_extension

import (
	"fmt"
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) getMyBlockList(err error, packet nex.PacketInterface, callID uint32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	// --- Debug Code Start ---
	common_globals.Logger.Info(fmt.Sprintf("--- Monitoring GetMyBlockList for PID: %d ---", connection.PID()))
	// --- Debug Code End ---

	commonProtocol.manager.Mutex.RLock()
	defer commonProtocol.manager.Mutex.RUnlock()

	blockList, nexError := database.GetBlockList(commonProtocol.manager, connection.PID())
	if nexError != nil {
		// --- Debug Code Start ---
		common_globals.Logger.Error(fmt.Sprintf("Error getting block list for PID %d: %s", connection.PID(), nexError.Error()))
		// --- Debug Code End ---
		return nil, nexError
	}

	// --- Debug Code Start ---
	common_globals.Logger.Info(fmt.Sprintf("Found block list for PID %d: %v", connection.PID(), blockList))
	// --- Debug Code End ---

	lstPrincipalID := types.NewList[types.PID]()
	for _, pid := range blockList {
		lstPrincipalID = append(lstPrincipalID, types.PID(pid))
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())
	lstPrincipalID.WriteTo(rmcResponseStream)
	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodGetMyBlockList
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterGetMyBlockList != nil {
		go commonProtocol.OnAfterGetMyBlockList(packet)
	}

	return rmcResponse, nil
}

