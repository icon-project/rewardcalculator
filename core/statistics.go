package core

import (
	"fmt"
	"github.com/icon-project/rewardcalculator/common/codec"
	"log"
	"reflect"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/oleiade/reflections"
)

type Statistics struct {
	Accounts uint64
	Beta1    common.HexInt
	Beta2    common.HexInt
	Beta3    common.HexInt
}

func (stats *Statistics) String() string {
	return fmt.Sprintf("==== Statistics - Accounts: %d, Beta1: %s, Beta2: %s, Beta3: %s",
		stats.Accounts,
		stats.Beta1.String(),
		stats.Beta2.String(),
		stats.Beta3.String())
}

func (stats *Statistics) Bytes() []byte {
	var bytes []byte
	if bs, err := codec.MarshalToBytes(stats); err != nil {
		log.Panicf("Failed to marshal Statistics %+v. err=%+v", stats, err)
		return nil
	} else {
		bytes = bs
	}
	return bytes
}

func (stats *Statistics) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, stats)
	if err != nil {
		return err
	}
	return nil
}

func (stats *Statistics) Set(field string, value interface{}) error {
	return reflections.SetField(stats, field, value)
}

func (stats *Statistics) Increase(field string, value interface{}) error {
	org, err := reflections.GetField(stats, field)
	if err != nil {
		return err
	}

	if reflect.TypeOf(org) != reflect.TypeOf(value) {
		return fmt.Errorf("provided value type didn't match field type")
	}

	switch v := value.(type) {
	case uint64:
		var newValue uint64
		newValue = org.(uint64) + v
		return stats.Set(field, newValue)
	case common.HexInt:
		var newValue common.HexInt
		o := org.(common.HexInt)
		newValue.Add(&o.Int, &v.Int)
		return stats.Set(field, newValue)
	}

	return nil
}

func (stats *Statistics) Decrease(field string, value interface{}) error {
	org, err := reflections.GetField(stats, field)
	if err != nil {
		return err
	}

	switch v := value.(type) {
	case uint64:
		var newValue uint64
		newValue = org.(uint64) - v
		return stats.Set(field, newValue)
	case common.HexInt:
		var newValue common.HexInt
		o := org.(common.HexInt)
		newValue.Sub(&o.Int, &v.Int)
		return stats.Set(field, newValue)
	}

	return nil
}
