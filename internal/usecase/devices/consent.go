package devices

import (
	"context"
	"strconv"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func (uc *UseCase) CancelUserConsent(c context.Context, guid string) (dto.UserConsentMessage, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return dto.UserConsentMessage{}, err
	}

	if item == nil || item.GUID == "" {
		return dto.UserConsentMessage{}, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	response, err := device.CancelUserConsentRequest()
	if err != nil {
		return dto.UserConsentMessage{}, err
	}

	headerDTO := dto.UserConsentHeader{
		To:          response.Header.To,
		Action:      response.Header.Action.Value,
		MessageID:   response.Header.MessageID,
		ResourceURI: response.Header.ResourceURI,
	}
	if response.Header.RelatesTo != 0 {
		headerDTO.RelatesTo = strconv.Itoa(response.Header.RelatesTo)
	}

	bodyDTO := dto.UserConsentBody{
		ReturnValue: response.Body.CancelOptInResponse.ReturnValue,
	}

	return dto.UserConsentMessage{
		Header: headerDTO,
		Body:   bodyDTO,
	}, nil
}

func (uc *UseCase) GetUserConsentCode(c context.Context, guid string) (dto.UserConsentMessage, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return dto.UserConsentMessage{}, err
	}

	if item == nil || item.GUID == "" {
		return dto.UserConsentMessage{}, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	response, err := device.GetUserConsentCode()
	if err != nil {
		return dto.UserConsentMessage{}, err
	}

	headerDTO := dto.UserConsentHeader{
		To:          response.Header.To,
		Action:      response.Header.Action.Value,
		MessageID:   response.Header.MessageID,
		ResourceURI: response.Header.ResourceURI,
	}
	if response.Header.RelatesTo != 0 {
		headerDTO.RelatesTo = strconv.Itoa(response.Header.RelatesTo)
	}

	bodyDTO := dto.UserConsentBody{
		ReturnValue: response.Body.StartOptInResponse.ReturnValue,
	}

	return dto.UserConsentMessage{
		Header: headerDTO,
		Body:   bodyDTO,
	}, nil
}

func (uc *UseCase) SendConsentCode(c context.Context, userConsent dto.UserConsentCode, guid string) (dto.UserConsentMessage, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return dto.UserConsentMessage{}, err
	}

	if item == nil || item.GUID == "" {
		return dto.UserConsentMessage{}, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	consentCode, _ := strconv.Atoi(userConsent.ConsentCode)

	response, err := device.SendConsentCode(consentCode)
	if err != nil {
		return dto.UserConsentMessage{}, err
	}

	headerDTO := dto.UserConsentHeader{
		To:          response.Header.To,
		Action:      response.Header.Action.Value,
		MessageID:   response.Header.MessageID,
		ResourceURI: response.Header.ResourceURI,
	}
	if response.Header.RelatesTo != 0 {
		headerDTO.RelatesTo = strconv.Itoa(response.Header.RelatesTo)
	}

	bodyDTO := dto.UserConsentBody{
		ReturnValue: response.Body.SendOptInCodeResponse.ReturnValue,
	}

	return dto.UserConsentMessage{
		Header: headerDTO,
		Body:   bodyDTO,
	}, nil
}
