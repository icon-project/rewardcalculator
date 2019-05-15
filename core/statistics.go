package core

import (
	"fmt"
	"reflect"

	"github.com/icon-project/rewardcalculator/common"
	"github.com/oleiade/reflections"
)

type statistics struct {
	Accounts uint64
	IScore   common.HexInt
}

func (stats *statistics) String() string {
	return fmt.Sprintf("Accounts: %d IScore: %s", stats.Accounts, stats.IScore.String())
}

func (stats *statistics) Set(field string, value interface{}) error {
	return reflections.SetField(stats, field, value)
}

func (stats *statistics) Increase(field string, value interface{}) error {
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

func (stats *statistics) Decrease(field string, value interface{}) error {
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
