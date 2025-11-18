// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package nroci

import (
	"fmt"
	"time"

	"github.com/oracle/nosql-go-sdk/nosqldb"
)

func init() {
	//add more here
}

type OCIClient interface {
	AddReplica(req *nosqldb.AddReplicaRequest) (*nosqldb.TableResult, error)
	Close() error
	Delete(req *nosqldb.DeleteRequest) (*nosqldb.DeleteResult, error)
	DoSystemRequest(req *nosqldb.SystemRequest) (*nosqldb.SystemResult, error)
	DoSystemRequestAndWait(statement string, timeout time.Duration, pollInterval time.Duration) (*nosqldb.SystemResult, error)
	DoTableRequest(req *nosqldb.TableRequest) (*nosqldb.TableResult, error)
	DoTableRequestAndWait(req *nosqldb.TableRequest, timeout time.Duration, pollInterval time.Duration) (*nosqldb.TableResult, error)
	DropReplica(req *nosqldb.DropReplicaRequest) (*nosqldb.TableResult, error)
	EnableRateLimiting(enable bool, usePercent float64)

	Get(req *nosqldb.GetRequest) (*nosqldb.GetResult, error)
	GetIndexes(req *nosqldb.GetIndexesRequest) (*nosqldb.GetIndexesResult, error)
	GetQueryVersion() int16
	GetReplicaStats(req *nosqldb.ReplicaStatsRequest) (*nosqldb.ReplicaStatsResult, error)
	GetSerialVersion() int16
	GetServerSerialVersion() int
	GetSystemStatus(req *nosqldb.SystemStatusRequest) (*nosqldb.SystemResult, error)
	GetTable(req *nosqldb.GetTableRequest) (*nosqldb.TableResult, error)
	GetTableUsage(req *nosqldb.TableUsageRequest) (*nosqldb.TableUsageResult, error)

	ListNamespaces() ([]string, error)
	ListRoles() ([]string, error)
	ListTables(req *nosqldb.ListTablesRequest) (*nosqldb.ListTablesResult, error)
	ListUsers() ([]nosqldb.UserInfo, error)

	MultiDelete(req *nosqldb.MultiDeleteRequest) (*nosqldb.MultiDeleteResult, error)
	Prepare(req *nosqldb.PrepareRequest) (*nosqldb.PrepareResult, error)
	Put(req *nosqldb.PutRequest) (*nosqldb.PutResult, error)
	Query(req *nosqldb.QueryRequest) (*nosqldb.QueryResult, error)
	ResetRateLimiters(tableName string)
	SetQueryVersion(qVer int16)
	SetSerialVersion(sVer int16)
	VerifyConnection() error
	WriteMultiple(req *nosqldb.WriteMultipleRequest) (*nosqldb.WriteMultipleResult, error)
}

type ConfigWrapper struct {
	Config *nosqldb.Config
}

type ClientWrapper struct {
	Client OCIClient
}

func NRDefaultConfig() *ConfigWrapper {
	cfg := nosqldb.Config{}
	return &ConfigWrapper{
		Config: &cfg,
	}
}

func NRCreateClient(cfg *ConfigWrapper) (*ClientWrapper, error) {
	client, err := nosqldb.NewClient(*cfg.Config)
	if err != nil {
		return nil, fmt.Errorf("error creating OCI Client: %s", err.Error())
	}
	return &ClientWrapper{
		Client: client,
	}, nil
}
