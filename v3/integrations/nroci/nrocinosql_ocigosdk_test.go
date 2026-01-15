package nroci

import (
	"testing"

	"github.com/oracle/oci-go-sdk/nosql"
)

func Test_extractRequestFieldsOCI(t *testing.T) {
	compartmentID := "ocid1.compartment.test"
	statement := "SELECT * FROM TABLE"
	tableName := "TABLE"
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		req   any
		want  string // collection
		want2 string // statement
		want3 string // compartmentID
	}{
		{
			name: "QueryRequest all details exist",
			req: nosql.QueryRequest{
				QueryDetails: nosql.QueryDetails{
					CompartmentId: &compartmentID,
					Statement:     &statement,
				},
			},
			want:  "",
			want2: "SELECT * FROM TABLE",
			want3: "ocid1.compartment.test",
		},
		{
			name: "QueryRequest all details are nil",
			req: nosql.QueryRequest{
				QueryDetails: nosql.QueryDetails{
					CompartmentId: nil,
					Statement:     nil,
				},
			},
			want:  "",
			want2: "",
			want3: "",
		},
		{
			name:  "QueryRequest is empty",
			req:   nosql.QueryRequest{},
			want:  "",
			want2: "",
			want3: "",
		},
		{
			name: "QueryRequest details are empty",
			req: nosql.QueryRequest{
				QueryDetails: nosql.QueryDetails{},
			},
			want:  "",
			want2: "",
			want3: "",
		},
		{
			name: "QueryRequest some details exist",
			req: nosql.QueryRequest{
				QueryDetails: nosql.QueryDetails{
					CompartmentId: nil,
					Statement:     &statement,
				},
			},
			want:  "",
			want2: "SELECT * FROM TABLE",
			want3: "",
		},
		{
			name: "UpdateRowRequest all details exist",
			req: nosql.UpdateRowRequest{
				TableNameOrId: &tableName,
				UpdateRowDetails: nosql.UpdateRowDetails{
					CompartmentId: &compartmentID,
				},
			},
			want:  "TABLE",
			want2: "",
			want3: "ocid1.compartment.test",
		},
		{
			name: "UpdateRowRequest no details exist",
			req: nosql.UpdateRowRequest{
				TableNameOrId: nil,
				UpdateRowDetails: nosql.UpdateRowDetails{
					CompartmentId: nil,
				},
			},
			want:  "",
			want2: "",
			want3: "",
		},
		{
			name:  "UpdateRowRequest is empty",
			req:   nosql.UpdateRowRequest{},
			want:  "",
			want2: "",
			want3: "",
		},
		{
			name: "UpdateRowRequest details are empty",
			req: nosql.UpdateRowRequest{
				UpdateRowDetails: nosql.UpdateRowDetails{},
			},
			want:  "",
			want2: "",
			want3: "",
		},
		{
			name: "UpdateRowRequest details are empty but TableNameOrId exists",
			req: nosql.UpdateRowRequest{
				TableNameOrId:    &tableName,
				UpdateRowDetails: nosql.UpdateRowDetails{},
			},
			want:  "TABLE",
			want2: "",
			want3: "",
		},
		{
			name: "UpdateRowRequest details exists but TableNameOrId is empty",
			req: nosql.UpdateRowRequest{
				TableNameOrId: nil,
				UpdateRowDetails: nosql.UpdateRowDetails{
					CompartmentId: &compartmentID,
				},
			},
			want:  "",
			want2: "",
			want3: "ocid1.compartment.test",
		},
		{
			name: "CreateTableRequest all details exist",
			req: nosql.CreateTableRequest{
				CreateTableDetails: nosql.CreateTableDetails{
					Name:          &tableName,
					CompartmentId: &compartmentID,
					DdlStatement:  &statement,
				},
			},
			want:  "TABLE",
			want2: "SELECT * FROM TABLE",
			want3: "ocid1.compartment.test",
		},
		{
			name: "CreateTableRequest no details exist",
			req: nosql.CreateTableRequest{
				CreateTableDetails: nosql.CreateTableDetails{
					Name:          nil,
					CompartmentId: nil,
					DdlStatement:  nil,
				},
			},
			want:  "",
			want2: "",
			want3: "",
		},
		{
			name:  "CreateTableRequest is empty",
			req:   nosql.CreateTableRequest{},
			want:  "",
			want2: "",
			want3: "",
		},
		{
			name: "CreateTableRequest details are empty",
			req: nosql.CreateTableRequest{
				CreateTableDetails: nosql.CreateTableDetails{},
			},
			want:  "",
			want2: "",
			want3: "",
		},
		{
			name: "CreateTableRequest some details exist",
			req: nosql.CreateTableRequest{
				CreateTableDetails: nosql.CreateTableDetails{
					Name:          &tableName,
					CompartmentId: nil,
					DdlStatement:  nil,
				},
			},
			want:  "TABLE",
			want2: "",
			want3: "",
		},
		{
			name: "DeleteRowRequest all details exist",
			req: nosql.DeleteRowRequest{
				TableNameOrId: &tableName,
				CompartmentId: &compartmentID,
			},
			want:  "TABLE",
			want2: "",
			want3: "ocid1.compartment.test",
		},
		{
			name: "DeleteRowRequest no details exist",
			req: nosql.DeleteRowRequest{
				TableNameOrId: nil,
				CompartmentId: nil,
			},
			want:  "",
			want2: "",
			want3: "",
		},
		{
			name:  "DeleteRowRequest is empty",
			req:   nosql.DeleteRowRequest{},
			want:  "",
			want2: "",
			want3: "",
		},
		{
			name: "DeleteRowRequest only TableNameOrId exists",
			req: nosql.DeleteRowRequest{
				TableNameOrId: &tableName,
				CompartmentId: nil,
			},
			want:  "TABLE",
			want2: "",
			want3: "",
		},
		{
			name: "DeleteRowRequest only CompartmentId exists",
			req: nosql.DeleteRowRequest{
				TableNameOrId: nil,
				CompartmentId: &compartmentID,
			},
			want:  "",
			want2: "",
			want3: "ocid1.compartment.test",
		},
		{
			name: "DeleteTableRequest all details exist",
			req: nosql.DeleteTableRequest{
				TableNameOrId: &tableName,
				CompartmentId: &compartmentID,
			},
			want:  "TABLE",
			want2: "",
			want3: "ocid1.compartment.test",
		},
		{
			name: "DeleteTableRequest no details exist",
			req: nosql.DeleteTableRequest{
				TableNameOrId: nil,
				CompartmentId: nil,
			},
			want:  "",
			want2: "",
			want3: "",
		},
		{
			name:  "DeleteTableRequest is empty",
			req:   nosql.DeleteTableRequest{},
			want:  "",
			want2: "",
			want3: "",
		},
		{
			name: "DeleteTableRequest only TableNameOrId exists",
			req: nosql.DeleteTableRequest{
				TableNameOrId: &tableName,
				CompartmentId: nil,
			},
			want:  "TABLE",
			want2: "",
			want3: "",
		},
		{
			name: "DeleteTableRequest only CompartmentId exists",
			req: nosql.DeleteTableRequest{
				TableNameOrId: nil,
				CompartmentId: &compartmentID,
			},
			want:  "",
			want2: "",
			want3: "ocid1.compartment.test",
		},
		{
			name: "GetRowRequest all details exist",
			req: nosql.GetRowRequest{
				TableNameOrId: &tableName,
				CompartmentId: &compartmentID,
			},
			want:  "TABLE",
			want2: "",
			want3: "ocid1.compartment.test",
		},
		{
			name: "GetRowRequest no details exist",
			req: nosql.GetRowRequest{
				TableNameOrId: nil,
				CompartmentId: nil,
			},
			want:  "",
			want2: "",
			want3: "",
		},
		{
			name:  "GetRowRequest is empty",
			req:   nosql.GetRowRequest{},
			want:  "",
			want2: "",
			want3: "",
		},
		{
			name: "GetRowRequest only TableNameOrId exists",
			req: nosql.GetRowRequest{
				TableNameOrId: &tableName,
				CompartmentId: nil,
			},
			want:  "TABLE",
			want2: "",
			want3: "",
		},
		{
			name: "GetRowRequest only CompartmentId exists",
			req: nosql.GetRowRequest{
				TableNameOrId: nil,
				CompartmentId: &compartmentID,
			},
			want:  "",
			want2: "",
			want3: "ocid1.compartment.test",
		},
		{
			name: "GetTableRequest all details exist",
			req: nosql.GetTableRequest{
				TableNameOrId: &tableName,
				CompartmentId: &compartmentID,
			},
			want:  "TABLE",
			want2: "",
			want3: "ocid1.compartment.test",
		},
		{
			name: "GetTableRequest no details exist",
			req: nosql.GetTableRequest{
				TableNameOrId: nil,
				CompartmentId: nil,
			},
			want:  "",
			want2: "",
			want3: "",
		},
		{
			name:  "GetTableRequest is empty",
			req:   nosql.GetTableRequest{},
			want:  "",
			want2: "",
			want3: "",
		},
		{
			name: "GetTableRequest only TableNameOrId exists",
			req: nosql.GetTableRequest{
				TableNameOrId: &tableName,
				CompartmentId: nil,
			},
			want:  "TABLE",
			want2: "",
			want3: "",
		},
		{
			name: "GetTableRequest only CompartmentId exists",
			req: nosql.GetTableRequest{
				TableNameOrId: nil,
				CompartmentId: &compartmentID,
			},
			want:  "",
			want2: "",
			want3: "ocid1.compartment.test",
		},
		{
			name: "UpdateTableRequest all details exist",
			req: nosql.UpdateTableRequest{
				TableNameOrId: &tableName,
				UpdateTableDetails: nosql.UpdateTableDetails{
					CompartmentId: &compartmentID,
					DdlStatement:  &statement,
				},
			},
			want:  "TABLE",
			want2: "SELECT * FROM TABLE",
			want3: "ocid1.compartment.test",
		},
		{
			name: "UpdateTableRequest no details exist",
			req: nosql.UpdateTableRequest{
				TableNameOrId: nil,
				UpdateTableDetails: nosql.UpdateTableDetails{
					CompartmentId: nil,
					DdlStatement:  nil,
				},
			},
			want:  "",
			want2: "",
			want3: "",
		},
		{
			name:  "UpdateTableRequest is empty",
			req:   nosql.UpdateTableRequest{},
			want:  "",
			want2: "",
			want3: "",
		},
		{
			name: "UpdateTableRequest details are empty",
			req: nosql.UpdateTableRequest{
				UpdateTableDetails: nosql.UpdateTableDetails{},
			},
			want:  "",
			want2: "",
			want3: "",
		},
		{
			name: "UpdateTableRequest details are empty but TableNameOrId exists",
			req: nosql.UpdateTableRequest{
				TableNameOrId:      &tableName,
				UpdateTableDetails: nosql.UpdateTableDetails{},
			},
			want:  "TABLE",
			want2: "",
			want3: "",
		},
		{
			name: "UpdateTableRequest details exist but TableNameOrId is empty",
			req: nosql.UpdateTableRequest{
				TableNameOrId: nil,
				UpdateTableDetails: nosql.UpdateTableDetails{
					CompartmentId: &compartmentID,
					DdlStatement:  &statement,
				},
			},
			want:  "",
			want2: "SELECT * FROM TABLE",
			want3: "ocid1.compartment.test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got2, got3 := extractRequestFieldsOCI(tt.req)
			if got != tt.want {
				t.Errorf("extractRequestFieldsOCI() = %v, want %v", got, tt.want)
			}
			if got2 != tt.want2 {
				t.Errorf("extractRequestFieldsOCI() = %v, want %v", got2, tt.want2)
			}
			if got3 != tt.want3 {
				t.Errorf("extractRequestFieldsOCI() = %v, want %v", got3, tt.want3)
			}
		})
	}
}
