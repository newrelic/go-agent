// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cqlparse

import (
	"testing"

	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func TestParseQuery(t *testing.T) {
	tests := []struct {
		// Named input parameters for target function.
		query string

		expectedCollection string
		expectedOperation  string
	}{
		{
			query:              "SELECT * FROM table",
			expectedCollection: "table",
			expectedOperation:  "select",
		},
		{
			query:              "SELECT name, occupation FROM users;",
			expectedCollection: "users",
			expectedOperation:  "select",
		},
		{
			query:              "SELECT name, occupation FROM users",
			expectedCollection: "users",
			expectedOperation:  "select",
		},
		{
			query:              "SELECT name, occupation FROM users WHERE userid IN (199, 200, 207);",
			expectedCollection: "users",
			expectedOperation:  "select",
		},
		{
			query:              "SELECT JSON name, occupation FROM users WHERE userid IN (199, 200, 207);",
			expectedCollection: "users",
			expectedOperation:  "select",
		},
		{
			query:              "SELECT name AS user_name, occupation AS user_occupation FROM users;",
			expectedCollection: "users",
			expectedOperation:  "select",
		},
		// SELECT — additional cases from https://cassandra.apache.org/doc/4.0/cassandra/cql/dml.html
		{
			query:              "SELECT COUNT(*) AS user_count FROM users;",
			expectedCollection: "users",
			expectedOperation:  "select",
		},
		{
			query:              "SELECT intAsBlob(4) AS four FROM t;",
			expectedCollection: "t",
			expectedOperation:  "select",
		},
		{
			query: `SELECT time, value
FROM events
WHERE event_type = 'myEvent'
  AND time > '2011-02-03'
  AND time <= '2012-01-01';`,
			expectedCollection: "events",
			expectedOperation:  "select",
		},
		{
			query: `SELECT entry_title, content FROM posts
 WHERE userid = 'john doe'
   AND blog_title = 'John''s Blog'
   AND posted_at >= '2012-01-01' AND posted_at < '2012-01-31';`,
			expectedCollection: "posts",
			expectedOperation:  "select",
		},
		{
			query:              "SELECT * FROM posts WHERE token(userid) > token('tom') AND token(userid) < token('bob');",
			expectedCollection: "posts",
			expectedOperation:  "select",
		},
		{
			query: `SELECT * FROM posts
 WHERE userid = 'john doe'
   AND (blog_title, posted_at) > ('John''s Blog', '2012-01-01');`,
			expectedCollection: "posts",
			expectedOperation:  "select",
		},
		{
			query: `SELECT * FROM posts
 WHERE userid = 'john doe'
   AND (blog_title, posted_at) IN (('John''s Blog', '2012-01-01'), ('Extreme Chess', '2014-06-01'));`,
			expectedCollection: "posts",
			expectedOperation:  "select",
		},
		{
			query:              "SELECT * FROM users WHERE birth_year = 1981 AND country = 'FR' ALLOW FILTERING;",
			expectedCollection: "users",
			expectedOperation:  "select",
		},
		// SELECT — mixed case
		{
			query:              "select name, occupation from users;",
			expectedCollection: "users",
			expectedOperation:  "select",
		},
		{
			query:              "Select * From users;",
			expectedCollection: "users",
			expectedOperation:  "select",
		},
		// SELECT — line comments
		{
			query: `SELECT -- * FROM tricky
* FROM users`,
			expectedCollection: "users",
			expectedOperation:  "select",
		},
		{
			query: `    -- SELECT * FROM tricky
SELECT * FROM users`,
			expectedCollection: "users",
			expectedOperation:  "select",
		},
		{
			query: `SELECT * FROM -- tricky
users`,
			expectedCollection: "users",
			expectedOperation:  "select",
		},
		// SELECT — block comments
		{
			query:              `/* find all users */ SELECT * FROM users;`,
			expectedCollection: "users",
			expectedOperation:  "select",
		},
		{
			query: `SELECT /* columns */ name, occupation
FROM /* table */ users;`,
			expectedCollection: "users",
			expectedOperation:  "select",
		},
		{
			query: `/* multi
line
comment */
SELECT * FROM users;`,
			expectedCollection: "users",
			expectedOperation:  "select",
		},
		// SELECT — semicolon/whitespace prefixes
		{
			query:              ";SELECT * FROM users;",
			expectedCollection: "users",
			expectedOperation:  "select",
		},
		{
			query:              "  ;;  ; SELECT * FROM users;",
			expectedCollection: "users",
			expectedOperation:  "select",
		},
		{
			query: ` ;
SELECT * FROM users`,
			expectedCollection: "users",
			expectedOperation:  "select",
		},
		// SELECT — keyspace-qualified
		{
			query:              "SELECT * FROM mykeyspace.users;",
			expectedCollection: "users",
			expectedOperation:  "select",
		},
		// INSERT — from https://cassandra.apache.org/doc/4.0/cassandra/cql/dml.html
		{
			query: `INSERT INTO NerdMovies (movie, director, main_actor, year)
   VALUES ('Serenity', 'Joss Whedon', 'Nathan Fillion', 2005)
   USING TTL 86400;`,
			expectedCollection: "NerdMovies",
			expectedOperation:  "insert",
		},
		{
			query:              `INSERT INTO NerdMovies JSON '{"movie": "Serenity", "director": "Joss Whedon", "year": 2005}';`,
			expectedCollection: "NerdMovies",
			expectedOperation:  "insert",
		},
		{
			query:              "INSERT INTO users (userid, password, name) VALUES ('user2', 'ch@ngem3b', 'second user') IF NOT EXISTS;",
			expectedCollection: "users",
			expectedOperation:  "insert",
		},
		{
			query:              "INSERT INTO users (userid, password) VALUES ('user4', 'ch@ngem3c');",
			expectedCollection: "users",
			expectedOperation:  "insert",
		},
		// INSERT — mixed case
		{
			query:              "insert into users (userid) values ('u1');",
			expectedCollection: "users",
			expectedOperation:  "insert",
		},
		{
			query:              "Insert Into users (userid) Values ('u1');",
			expectedCollection: "users",
			expectedOperation:  "insert",
		},
		// INSERT — line comment
		{
			query: `-- INSERT INTO tricky (x) VALUES (1)
INSERT INTO users (userid) VALUES ('u1')`,
			expectedCollection: "users",
			expectedOperation:  "insert",
		},
		// INSERT — block comment
		{
			query:              `/* INSERT INTO tricky */ INSERT INTO users (userid) VALUES ('u1');`,
			expectedCollection: "users",
			expectedOperation:  "insert",
		},
		// INSERT — semicolon prefix
		{
			query:              "   INSERT INTO users (userid) VALUES ('u1');",
			expectedCollection: "users",
			expectedOperation:  "insert",
		},
		// INSERT — keyspace-qualified
		{
			query:              "INSERT INTO mykeyspace.NerdMovies (movie) VALUES ('Serenity');",
			expectedCollection: "NerdMovies",
			expectedOperation:  "insert",
		},
		// UPDATE — from https://cassandra.apache.org/doc/4.0/cassandra/cql/dml.html
		{
			query: `UPDATE NerdMovies USING TTL 400
   SET director   = 'Joss Whedon',
       main_actor = 'Nathan Fillion',
       year       = 2005
 WHERE movie = 'Serenity';`,
			expectedCollection: "NerdMovies",
			expectedOperation:  "update",
		},
		{
			query: `UPDATE UserActions
   SET total = total + 2
   WHERE user = B70DE1D0-9908-4AE3-BE34-5573E5B09F14
     AND action = 'click';`,
			expectedCollection: "UserActions",
			expectedOperation:  "update",
		},
		{
			query:              "UPDATE users SET password = 'ps22dhds' WHERE userid = 'user3';",
			expectedCollection: "users",
			expectedOperation:  "update",
		},
		// UPDATE — mixed case
		{
			query:              "update users set name = 'foo' where userid = 'u1';",
			expectedCollection: "users",
			expectedOperation:  "update",
		},
		// UPDATE — line comment
		{
			query: `UPDATE -- NerdMovies
users SET name = 'foo' WHERE userid = 'u1';`,
			expectedCollection: "users",
			expectedOperation:  "update",
		},
		// UPDATE — block comment
		{
			query: `UPDATE users /* USING TTL 400 */
SET name = 'foo'
WHERE userid = 'u1';`,
			expectedCollection: "users",
			expectedOperation:  "update",
		},
		// UPDATE — keyspace-qualified
		{
			query:              "UPDATE mykeyspace.users SET name = 'foo' WHERE userid = 'u1';",
			expectedCollection: "users",
			expectedOperation:  "update",
		},
		// DELETE — from https://cassandra.apache.org/doc/4.0/cassandra/cql/dml.html
		{
			query: `DELETE FROM NerdMovies USING TIMESTAMP 1240003134
 WHERE movie = 'Serenity';`,
			expectedCollection: "NerdMovies",
			expectedOperation:  "delete",
		},
		{
			query: `DELETE phone FROM Users
 WHERE userid IN (C73DE1D3-AF08-40F3-B124-3FF3E5109F22, B70DE1D0-9908-4AE3-BE34-5573E5B09F14);`,
			expectedCollection: "Users",
			expectedOperation:  "delete",
		},
		{
			query:              "DELETE name FROM users WHERE userid = 'user1';",
			expectedCollection: "users",
			expectedOperation:  "delete",
		},
		// DELETE — mixed case
		{
			query:              "delete from users where userid = 'u1';",
			expectedCollection: "users",
			expectedOperation:  "delete",
		},
		// DELETE — line comment
		{
			query: `DELETE -- phone
FROM users WHERE userid = 'u1';`,
			expectedCollection: "users",
			expectedOperation:  "delete",
		},
		// DELETE — block comment
		{
			query:              `/* comment */ DELETE FROM users WHERE userid = 'u1';`,
			expectedCollection: "users",
			expectedOperation:  "delete",
		},
		// DELETE — keyspace-qualified
		{
			query:              "DELETE FROM mykeyspace.users WHERE userid = 'u1';",
			expectedCollection: "users",
			expectedOperation:  "delete",
		},
		// TRUNCATE
		{
			query:              "TRUNCATE users;",
			expectedCollection: "users",
			expectedOperation:  "truncate",
		},
		{
			query:              "TRUNCATE TABLE users;",
			expectedCollection: "users",
			expectedOperation:  "truncate",
		},
		{
			query:              "truncate table NerdMovies;",
			expectedCollection: "NerdMovies",
			expectedOperation:  "truncate",
		},
		{
			query:              "TRUNCATE TABLE mykeyspace.users;",
			expectedCollection: "users",
			expectedOperation:  "truncate",
		},
		// BATCH — from https://cassandra.apache.org/doc/4.0/cassandra/cql/dml.html
		{
			query: `BEGIN BATCH
   INSERT INTO users (userid, password, name) VALUES ('user2', 'ch@ngem3b', 'second user');
   UPDATE users SET password = 'ps22dhds' WHERE userid = 'user3';
   INSERT INTO users (userid, password) VALUES ('user4', 'ch@ngem3c');
   DELETE name FROM users WHERE userid = 'user1';
APPLY BATCH;`,
			expectedCollection: "",
			expectedOperation:  "begin",
		},
		{
			query:              "BEGIN UNLOGGED BATCH",
			expectedCollection: "",
			expectedOperation:  "begin",
		},
		{
			query:              "BEGIN COUNTER BATCH",
			expectedCollection: "",
			expectedOperation:  "begin",
		},
		{
			query:              "begin batch",
			expectedCollection: "",
			expectedOperation:  "begin",
		},
		{
			query:              "APPLY BATCH;",
			expectedCollection: "",
			expectedOperation:  "apply",
		},
		{
			query:              "apply batch;",
			expectedCollection: "",
			expectedOperation:  "apply",
		},
		// DDL — operation detected, no collection extracted
		{
			query:              "CREATE TABLE users (userid text PRIMARY KEY, name text);",
			expectedCollection: "",
			expectedOperation:  "create",
		},
		{
			query:              "CREATE KEYSPACE mykeyspace WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};",
			expectedCollection: "",
			expectedOperation:  "create",
		},
		{
			query:              "CREATE INDEX ON users(birth_year);",
			expectedCollection: "",
			expectedOperation:  "create",
		},
		{
			query:              "DROP TABLE users;",
			expectedCollection: "",
			expectedOperation:  "drop",
		},
		{
			query:              "DROP KEYSPACE mykeyspace;",
			expectedCollection: "",
			expectedOperation:  "drop",
		},
		{
			query:              "ALTER TABLE users ADD email text;",
			expectedCollection: "",
			expectedOperation:  "alter",
		},
		{
			query:              "ALTER TABLE users DROP email;",
			expectedCollection: "",
			expectedOperation:  "alter",
		},
		{
			query:              "USE mykeyspace;",
			expectedCollection: "",
			expectedOperation:  "use",
		},
		// Unknown operations
		{
			query:              "",
			expectedCollection: "",
			expectedOperation:  "other",
		},
		{
			query:              "EXPLAIN SELECT * FROM users;",
			expectedCollection: "",
			expectedOperation:  "other",
		},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			seg := &newrelic.DatastoreSegment{}
			ParseQuery(seg, tt.query)
			if tt.expectedOperation == "other" {
				// Allow for matching of Operation "other" to ""
				if seg.Operation != "" {
					t.Errorf("operation mismatch query='%s' wanted='%s' got='%s'",
						tt.query, tt.expectedOperation, seg.Operation)
				}
			} else if seg.Operation != tt.expectedOperation {
				t.Errorf("operation mismatch query='%s' wanted='%s' got='%s'",
					tt.query, tt.expectedOperation, seg.Operation)
			}
			// The Go agent subquery behavior does not matth the PHP Agent.
			if tt.expectedCollection == "(subquery)" {
				return
			}
			if tt.expectedCollection != seg.Collection {
				t.Errorf("table mismatch query='%s' wanted='%s' got='%s'",
					tt.query, tt.expectedCollection, seg.Collection)
			}
		})
	}
}
