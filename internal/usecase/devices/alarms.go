package devices

import (
	"context"
	"strconv"
	"strings"
	"time"

	amtAlarmClock "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/alarmclock"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/alarmclock"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

const (
	minutesPerDay  = 24 * 60
	minutesPerHour = 60
)

func (uc *UseCase) GetAlarmOccurrences(c context.Context, guid string) ([]dto.AlarmClockOccurrence, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return nil, err
	}

	if item == nil || item.GUID == "" {
		return nil, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	alarms, err := device.GetAlarmOccurrences()
	if err != nil {
		return nil, err
	}

	if alarms == nil {
		alarms = []alarmclock.AlarmClockOccurrence{}
	}

	// iterate over the data and convert each entity to dto
	d1 := make([]dto.AlarmClockOccurrence, len(alarms))

	for i := range alarms {
		tmpEntity := alarms[i] // create a new variable to avoid memory aliasing
		d1[i] = *uc.alarmOccurrenceEntityToDTO(&tmpEntity)
	}

	return d1, nil
}

func (uc *UseCase) CreateAlarmOccurrences(c context.Context, guid string, alarm dto.AlarmClockOccurrenceInput) (dto.AddAlarmOutput, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return dto.AddAlarmOutput{}, err
	}

	if item == nil || item.GUID == "" {
		return dto.AddAlarmOutput{}, ErrNotFound
	}

	alarm.InstanceID = alarm.ElementName

	device := uc.device.SetupWsmanClient(*item, false, true)

	alarmReference, err := device.CreateAlarmOccurrences(alarm.InstanceID, alarm.StartTime, alarm.Interval, alarm.DeleteOnCompletion)
	if err != nil {
		return dto.AddAlarmOutput{}, ErrAMT.Wrap("CreateAlarmOccurrences", "device.CreateAlarmOccurrences", err)
	}

	d1 := *uc.addAlarmOutputEntityToDTO(&alarmReference)

	return d1, nil
}

func (uc *UseCase) DeleteAlarmOccurrences(c context.Context, guid, instanceID string) error {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return err
	}

	if item == nil || item.GUID == "" {
		return ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	err = device.DeleteAlarmOccurrences(instanceID)
	if err != nil {
		return err
	}

	return nil
}

func (uc *UseCase) addAlarmOutputEntityToDTO(d *amtAlarmClock.AddAlarmOutput) *dto.AddAlarmOutput {
	d1 := &dto.AddAlarmOutput{
		ReturnValue: int(d.ReturnValue),
	}

	return d1
}

func (uc *UseCase) alarmOccurrenceEntityToDTO(d *alarmclock.AlarmClockOccurrence) *dto.AlarmClockOccurrence {
	intervalInMinutes, _ := ParseInterval(d.Interval.Interval)
	interval, _ := strconv.Atoi(d.Interval.Interval)
	d1 := &dto.AlarmClockOccurrence{
		ElementName:        d.ElementName,
		InstanceID:         d.InstanceID,
		StartTime:          dto.StartTime{Datetime: d.StartTime.Datetime},
		Interval:           interval,
		IntervalInMinutes:  intervalInMinutes,
		DeleteOnCompletion: d.DeleteOnCompletion,
	}

	return d1
}

func ParseInterval(duration string) (int, error) {
	if duration == "" {
		return 0, nil
	}

	totalMinutes := 0

	// parse days
	duration = strings.TrimPrefix(duration, "P")
	indexOfD := strings.Index(duration, "D")

	if indexOfD != -1 {
		days, err := strconv.Atoi(duration[:indexOfD])
		if err != nil {
			return 0, err
		}

		totalMinutes = days * minutesPerDay
	}

	// parse time
	indexOfT := strings.Index(duration, "T")
	if indexOfT != -1 {
		duration = strings.ToLower(duration[indexOfT+1:])

		timeDuration, err := time.ParseDuration(duration)
		if err != nil {
			return 0, err
		}

		return totalMinutes + int(timeDuration.Minutes()), nil
	}

	return totalMinutes, nil
}
