package nrgorm

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/newrelic/go-agent/v3/newrelic"
	"gorm.io/gorm"
)

const startTimeKey = "newrelic_start_time"

type APMPlugin struct{}

var _ gorm.Plugin = (*APMPlugin)(nil)

func (p APMPlugin) Name() string {
	return "newrelic"
}

func (p APMPlugin) Initialize(db *gorm.DB) error {
	err := db.Callback().Create().Before("gorm:create").Register("newrelic:before_create", beforeCallback)
	if err != nil {
		return err
	}
	err = db.Callback().Query().Before("gorm:query").Register("newrelic:before_query", beforeCallback)
	if err != nil {
		return err
	}
	err = db.Callback().Update().Before("gorm:update").Register("newrelic:before_update", beforeCallback)
	if err != nil {
		return err
	}
	err = db.Callback().Delete().Before("gorm:delete").Register("newrelic:before_delete", beforeCallback)
	if err != nil {
		return err
	}

	err = db.Callback().Create().After("gorm:create").Register("newrelic:after_create", afterCallback("INSERT"))
	if err != nil {
		return err
	}
	err = db.Callback().Query().After("gorm:query").Register("newrelic:after_query", afterCallback("SELECT"))
	if err != nil {
		return err
	}
	err = db.Callback().Update().After("gorm:update").Register("newrelic:after_update", afterCallback("UPDATE"))
	if err != nil {
		return err
	}
	err = db.Callback().Delete().After("gorm:delete").Register("newrelic:after_delete", afterCallback("DELETE"))
	if err != nil {
		return err
	}

	return nil
}

func beforeCallback(db *gorm.DB) {
	if txn := newrelic.FromContext(db.Statement.Context); txn != nil {
		db.Set(startTimeKey, txn.StartSegmentNow())
	}
}

func afterCallback(operation string) func(db *gorm.DB) {
	return func(db *gorm.DB) {
		if startTime, ok := db.Get(startTimeKey); ok {
			segment := newrelic.DatastoreSegment{
				Product:            newrelic.DatastorePostgres,
				Collection:         db.Statement.Table,
				Operation:          operation,
				StartTime:          startTime.(newrelic.SegmentStartTime),
				ParameterizedQuery: db.Statement.SQL.String(),
				QueryParameters:    parseVars(db.Statement.Vars),
			}
			segment.End()
		}
	}
}

func parseVars(vars []interface{}) map[string]interface{} {
	queryParameters := make(map[string]interface{})
	for i, v := range vars {
		i := i
		v := v
		val := reflect.ValueOf(v)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		queryParameters[fmt.Sprintf("$%v", strconv.Itoa(i+1))] = fmt.Sprintf("%v", val.Interface())
	}
	return queryParameters
}
