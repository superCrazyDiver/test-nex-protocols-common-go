package secureconnection

import (
	"net"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/constants"
	"github.com/PretendoNetwork/nex-go/v2/types"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/v2/secure-connection"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func (commonProtocol *CommonProtocol) registerEx(err error, packet nex.PacketInterface, callID uint32, vecMyURLs types.List[types.StationURL], hCustomData types.DataHolder) (*nex.RMCMessage, *nex.Error) {
	if commonProtocol.ValidateLoginData == nil {
		common_globals.Logger.Error("SecureConnection::RegisterEx missing ValidateLoginData!")
		return nil, nex.NewError(nex.ResultCodes.Core.NotImplemented, "SecureConnection::RegisterEx missing ValidateLoginData!")
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint()

	var retval types.QResult
	pidConnectionID := types.NewUInt32(0)
	urlPublic := types.NewString("")

	errorCode := commonProtocol.ValidateLoginData(connection.PID(), hCustomData)
	if errorCode != nil {
		common_globals.Logger.Error(errorCode.Message)
		retval = types.NewQResultError(errorCode.ResultCode)
	} else {
		// * vecMyURLs may contain multiple StationURLs. Search them all
		var localStation *types.StationURL
		var publicStation *types.StationURL

		for _, stationURL := range vecMyURLs {
			natf, ok := stationURL.NATFiltering()
			if !ok {
				continue
			}

			natm, ok := stationURL.NATMapping()
			if !ok {
				continue
			}

			// * Station reports itself as being non-public (local)
			if localStation == nil && !stationURL.IsPublic() {
				localStation = &stationURL
			}

			// * Still did not find the station, trying heuristics
			if localStation == nil && natf == constants.UnknownNATFiltering && natm == constants.UnknownNATMapping {
				localStation = &stationURL
			}

			if publicStation == nil && stationURL.IsPublic() {
				publicStation = &stationURL
			}
		}

		if localStation == nil {
			common_globals.Logger.Error("Failed to find local station")
			return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
		}

		if publicStation == nil {
			publicStation = localStation

			var address string
			var port uint16

			// * We have to duplicate this because Go automatically breaks on switch statements
			switch clientAddress := connection.Address().(type) {
			case *net.UDPAddr:
				address = clientAddress.IP.String()
				port = uint16(clientAddress.Port)
			case *net.TCPAddr:
				address = clientAddress.IP.String()
				port = uint16(clientAddress.Port)
			}

			publicStation.SetAddress(address)
			publicStation.SetPortNumber(port)
			publicStation.SetNATFiltering(constants.UnknownNATFiltering)
			publicStation.SetNATMapping(constants.UnknownNATMapping)
			publicStation.SetType(uint8(constants.StationURLFlagPublic) | uint8(constants.StationURLFlagBehindNAT))
		}

		localStation.SetPrincipalID(connection.PID())
		publicStation.SetPrincipalID(connection.PID())

		localStation.SetRVConnectionID(connection.ID)
		publicStation.SetRVConnectionID(connection.ID)

		connection.StationURLs = append(connection.StationURLs, *localStation)
		connection.StationURLs = append(connection.StationURLs, *publicStation)

		retval = types.NewQResultSuccess(nex.ResultCodes.Core.Unknown)
		pidConnectionID = types.NewUInt32(connection.ID)
		urlPublic = types.NewString(publicStation.URL())
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	retval.WriteTo(rmcResponseStream)
	pidConnectionID.WriteTo(rmcResponseStream)
	urlPublic.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = secure_connection.ProtocolID
	rmcResponse.MethodID = secure_connection.MethodRegisterEx
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterRegisterEx != nil {
		go commonProtocol.OnAfterRegisterEx(packet, vecMyURLs, hCustomData)
	}

	return rmcResponse, nil
}
