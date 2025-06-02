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

	return response, nil
}

func (uc *UseCase) GetUserConsentCode(c context.Context, guid string) (dto.GetUserConsentMessage, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return dto.GetUserConsentMessage{}, err
	}

	if item == nil || item.GUID == "" {
		return dto.GetUserConsentMessage{}, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	code, err := device.GetUserConsentCode()
	if err != nil {
		return dto.GetUserConsentMessage{}, err
	}

	response := dto.GetUserConsentMessage{
		Body: dto.UserConsentMessage{
			Name:        code.XMLName,
			ReturnValue: code.ReturnValue,
		},
	}

	return response, nil
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

	return response, nil
}
