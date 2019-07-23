package collector

import (
	"fmt"
	"strconv"

	"github.com/uol/gobol"
	"github.com/uol/mycenae/lib/utils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	makePacketLogFields = []zapcore.Field{
		zap.String("package", "collector"),
		zap.String("func", "makePacket"),
	}

	errGenericValidation = errValidation("error in point validation")
)

// logPointError - logs the point error
func (collector *Collector) logPointError(point *TSDBpoint, err gobol.Error) {

	gblog.Warn(fmt.Sprintf("point validation error: %+v (%s)", *point, err.Error()), makePacketLogFields...)
}

// MakePacket - validates a point and fills the packet
func (collector *Collector) MakePacket(rcvMsg *TSDBpoint, number bool) (*Point, gobol.Error) {

	if number {
		if rcvMsg.Value == nil {
			if collector.settings.SilencePointValidationErrors {
				return nil, errGenericValidation
			}
			err := errValidation(`Wrong Format: Field "value" is required. NO information will be saved`)
			collector.logPointError(rcvMsg, err)
			return nil, err
		}
	} else {
		if rcvMsg.Text == "" {
			if collector.settings.SilencePointValidationErrors {
				return nil, errGenericValidation
			}
			err := errValidation(`Wrong Format: Field "text" is required. NO information will be saved`)
			collector.logPointError(rcvMsg, err)
			return nil, err
		}

		if len(rcvMsg.Text) > 10000 {
			if collector.settings.SilencePointValidationErrors {
				return nil, errGenericValidation
			}
			err := errValidation(`Wrong Format: Field "text" can not have more than 10k`)
			collector.logPointError(rcvMsg, err)
			return nil, err
		}
	}

	lt := len(rcvMsg.Tags)

	if lt == 0 {
		if collector.settings.SilencePointValidationErrors {
			return nil, errGenericValidation
		}
		err := errValidation(`Wrong Format: At least one tag is required. NO information will be saved`)
		collector.logPointError(rcvMsg, err)
		return nil, err
	}

	if !collector.validKey.MatchString(rcvMsg.Metric) {
		if collector.settings.SilencePointValidationErrors {
			return nil, errGenericValidation
		}
		err := errValidation(
			fmt.Sprintf(
				`Wrong Format: Field "metric" (%s) is not well formed. NO information will be saved`,
				rcvMsg.Metric,
			),
		)
		collector.logPointError(rcvMsg, err)
		return nil, err
	}

	packet := &Point{}

	var ok bool
	if packet.Keyset, ok = rcvMsg.Tags["ksid"]; !ok {
		if collector.settings.SilencePointValidationErrors {
			return nil, errGenericValidation
		}
		err := errValidation(`Wrong Format: Tag "ksid" is required. NO information will be saved`)
		collector.logPointError(rcvMsg, err)
		return nil, err
	}

	if !collector.keySet.IsKeySetNameValid(packet.Keyset) {
		if collector.settings.SilencePointValidationErrors {
			return nil, errGenericValidation
		}
		err := errValidation(
			fmt.Sprintf(
				`Wrong Format: Field "ksid" (%s) is not well formed. NO information will be saved`,
				packet.Keyset,
			),
		)
		collector.logPointError(rcvMsg, err)
		return nil, err
	}

	if strTTL, ok := rcvMsg.Tags["ttl"]; !ok {
		packet.TTL = collector.settings.DefaultTTL
		lt++
	} else {
		ttl, err := strconv.Atoi(strTTL)
		if err != nil {
			if collector.settings.SilencePointValidationErrors {
				return nil, errGenericValidation
			}
			err := errValidation(`Wrong Format: Tag "ttl" must be a positive number. NO information will be saved`)
			collector.logPointError(rcvMsg, err)
			return nil, err
		}
		if _, ok := collector.keyspaceTTLMap[ttl]; !ok {
			ttl = collector.settings.DefaultTTL
		}
		packet.TTL = ttl
	}
	rcvMsg.Tags["ttl"] = strconv.Itoa(packet.TTL)

	if lt == 2 {
		if collector.settings.SilencePointValidationErrors {
			return nil, errGenericValidation
		}
		err := errValidation(`Wrong Format: At least one tag other than "ksid" and "ttl" is required. NO information will be saved`)
		collector.logPointError(rcvMsg, err)
		return nil, err
	}

	for k, v := range rcvMsg.Tags {
		if !collector.validKey.MatchString(k) {
			if collector.settings.SilencePointValidationErrors {
				return nil, errGenericValidation
			}
			err := errValidation(
				fmt.Sprintf(
					`Wrong Format: Tag key (%s) is not well formed. NO information will be saved`,
					k,
				),
			)
			collector.logPointError(rcvMsg, err)
			return nil, err
		}
		if !collector.validKey.MatchString(v) {
			if collector.settings.SilencePointValidationErrors {
				return nil, errGenericValidation
			}
			err := errValidation(
				fmt.Sprintf(
					`Wrong Format: Tag value (%s) is not well formed. NO information will be saved`,
					v,
				),
			)
			collector.logPointError(rcvMsg, err)
			return nil, err
		}
	}

	keySetExists, gerr := collector.persist.metaStorage.CheckKeySet(packet.Keyset)
	if gerr != nil {
		if collector.settings.SilencePointValidationErrors {
			return nil, errGenericValidation
		}
		err := errInternalServerError("makePacket", "error checking keyspace existence", gerr)
		collector.logPointError(rcvMsg, err)
		return nil, err
	}
	if !keySetExists {
		if collector.settings.SilencePointValidationErrors {
			return nil, errGenericValidation
		}
		err := errValidation("ksid \"" + packet.Keyset + "\" not exists. NO information will be saved")
		collector.logPointError(rcvMsg, err)
		return nil, err
	}

	if rcvMsg.Timestamp == 0 {
		packet.Timestamp = utils.GetTimeNoMillis()
	} else {
		truncated, err := utils.MilliToSeconds(rcvMsg.Timestamp)
		if err != nil {
			if collector.settings.SilencePointValidationErrors {
				return nil, errGenericValidation
			}
			err := errBadRequest("makePacket", err.Error(), err)
			collector.logPointError(rcvMsg, err)
			return nil, err
		}
		packet.Timestamp = truncated
	}

	packet.Number = number
	packet.Message = rcvMsg
	packet.ID = GenerateID(rcvMsg)
	if !number {
		packet.ID = fmt.Sprintf("T%v", packet.ID)
	}

	return packet, nil
}
