package dialog

import (
	"fmt"
	"github.com/mysterium/node/communication"

	log "github.com/cihub/seelog"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/communication/nats/discovery"
	"github.com/mysterium/node/identity"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
)

// NewDialogWaiter constructs new DialogWaiter which works thru NATS connection.
func NewDialogWaiter(address *discovery.AddressNATS, signer identity.Signer) *dialogWaiter {
	return &dialogWaiter{
		myAddress: address,
		mySigner:  signer,
		dialogs:   make([]communication.Dialog, 0),
	}
}

const waiterLogPrefix = "[NATS.DialogWaiter] "

type dialogWaiter struct {
	myAddress *discovery.AddressNATS
	mySigner  identity.Signer
	dialogs   []communication.Dialog
}

func (waiter *dialogWaiter) Start() (dto_discovery.Contact, error) {
	log.Info(waiterLogPrefix, fmt.Sprintf("Connecting to: %#v", waiter.myAddress))

	err := waiter.myAddress.Connect()
	if err != nil {
		return dto_discovery.Contact{}, fmt.Errorf("failed to start my connection. %s", waiter.myAddress)
	}

	return waiter.myAddress.GetContact(), nil
}

func (waiter *dialogWaiter) Stop() error {
	for _, dialog := range waiter.dialogs {
		dialog.Close()
	}
	waiter.myAddress.Disconnect()

	return nil
}

func (waiter *dialogWaiter) ServeDialogs(dialogHandler communication.DialogHandler) error {
	createDialog := func(request *dialogCreateRequest) (*dialogCreateResponse, error) {
		if request.PeerID == "" {
			return &responseInvalidIdentity, nil
		}
		peerID := identity.FromAddress(request.PeerID)

		dialog := waiter.newDialogToPeer(peerID, waiter.newCodecForPeer(peerID))
		err := dialogHandler.Handle(dialog)
		if err != nil {
			log.Error(waiterLogPrefix, fmt.Sprintf("Failed dialog from: '%s'. %s", request.PeerID, err))
			return &responseInternalError, nil
		}

		waiter.dialogs = append(waiter.dialogs, dialog)

		log.Info(waiterLogPrefix, fmt.Sprintf("Accepted dialog from: '%s'", request.PeerID))
		return &responseOK, nil
	}

	myCodec := NewCodecSecured(communication.NewCodecJSON(), waiter.mySigner, identity.NewVerifierSigned())
	myReceiver := nats.NewReceiver(waiter.myAddress.GetConnection(), myCodec, waiter.myAddress.GetTopic())

	subscribeError := myReceiver.Respond(&dialogCreateConsumer{createDialog})
	return subscribeError
}

func (waiter *dialogWaiter) newCodecForPeer(peerID identity.Identity) *codecSecured {

	return NewCodecSecured(
		communication.NewCodecJSON(),
		waiter.mySigner,
		identity.NewVerifierIdentity(peerID),
	)
}

func (waiter *dialogWaiter) newDialogToPeer(peerID identity.Identity, peerCodec *codecSecured) *dialog {
	subTopic := waiter.myAddress.GetTopic() + "." + peerID.Address

	return &dialog{
		peerID:   peerID,
		Sender:   nats.NewSender(waiter.myAddress.GetConnection(), peerCodec, subTopic),
		Receiver: nats.NewReceiver(waiter.myAddress.GetConnection(), peerCodec, subTopic),
	}
}
